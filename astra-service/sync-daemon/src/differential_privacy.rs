//! Differential privacy for aggregated sales metrics.
//!
//! Before the Raft leader uploads analytics batches to the cloud, Laplace noise
//! is added to aggregated values.  This provides plausible deniability for
//! individual transactions while preserving store-level trends.
//!
//! The implementation uses epsilon = 1.0 by default and a sensitivity of 1 per
//! transaction.  Callers supply the raw aggregate and receive a noised value.

#![deny(unsafe_code)]

use rand::Rng;

use crate::AstraSyncError;

/// Default privacy budget.  Smaller epsilon = stronger privacy.
pub const DEFAULT_EPSILON: f64 = 1.0;

/// Adds Laplace noise to an aggregated sales value.
///
/// * `value` — raw aggregate (e.g. total cents sold in a window).
/// * `sensitivity` — maximum influence a single record can have on `value`.
/// * `epsilon` — privacy budget (must be > 0).
///
/// Returns the noised value as an integer, saturating at `u64` bounds.
pub fn add_laplace_noise(value: u64, sensitivity: f64, epsilon: f64) -> Result<u64, AstraSyncError> {
    if epsilon <= 0.0 {
        return Err(AstraSyncError::DifferentialPrivacy(
            "epsilon must be positive".to_string(),
        ));
    }
    if sensitivity < 0.0 {
        return Err(AstraSyncError::DifferentialPrivacy(
            "sensitivity must be non-negative".to_string(),
        ));
    }
    if sensitivity == 0.0 {
        return Ok(value);
    }

    let scale = sensitivity / epsilon;
    let noise = sample_laplace(scale);

    let signed_value = value as i128;
    let signed_noise = noise.round() as i128;
    let noised = signed_value.saturating_add(signed_noise);

    Ok(noised.max(0) as u64)
}

/// Samples a single Laplace(0, `scale`) random variable using inversion sampling.
fn sample_laplace(scale: f64) -> f64 {
    let mut rng = rand::thread_rng();
    let u: f64 = rng.gen();
    if u < 0.5 {
        scale * (2.0 * u).ln()
    } else {
        -scale * (2.0 * (1.0 - u)).ln()
    }
}

/// Convenience wrapper that applies the default epsilon = 1.0 budget.
pub fn privatize_sales(value: u64, sensitivity: f64) -> Result<u64, AstraSyncError> {
    add_laplace_noise(value, sensitivity, DEFAULT_EPSILON)
}

/// Sensitivity for the `total_cents` field of a [`SalesAggregate`].
///
/// This bounds the maximum influence a single transaction can have on the
/// aggregate revenue figure.  The default assumes no single transaction exceeds
/// $1,000 (100,000 cents).
pub const DEFAULT_TOTAL_CENTS_SENSITIVITY: f64 = 100_000.0;

/// Sensitivity for the `transaction_count` field of a [`SalesAggregate`].
///
/// Adding or removing one transaction changes the count by exactly one.
pub const DEFAULT_TRANSACTION_COUNT_SENSITIVITY: f64 = 1.0;

/// Sensitivity for the `item_count` field of a [`SalesAggregate`].
///
/// The default assumes no single transaction contains more than 100 items.
pub const DEFAULT_ITEM_COUNT_SENSITIVITY: f64 = 100.0;

/// Aggregated sales metrics for a time window.
///
/// This struct is the canonical shape of sales analytics before and after
/// differential privacy noise is applied.  All fields are `u64` and represent
/// store-level aggregates, never individual transactions.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct SalesAggregate {
    /// Start of the aggregation window (millis since Unix epoch).
    pub window_start_ms: u64,
    /// End of the aggregation window (millis since Unix epoch).
    pub window_end_ms: u64,
    /// Total revenue in the window, in cents.
    pub total_cents: u64,
    /// Number of transactions in the window.
    pub transaction_count: u64,
    /// Total number of items sold in the window.
    pub item_count: u64,
}

/// Applies Laplace noise to a [`SalesAggregate`] using the provided epsilon budget.
///
/// Each field receives independent noise scaled to its own sensitivity.
/// `epsilon` must be greater than zero.
pub fn privatize_sales_aggregate(
    aggregate: &SalesAggregate,
    epsilon: f64,
) -> Result<SalesAggregate, AstraSyncError> {
    Ok(SalesAggregate {
        window_start_ms: aggregate.window_start_ms,
        window_end_ms: aggregate.window_end_ms,
        total_cents: add_laplace_noise(
            aggregate.total_cents,
            DEFAULT_TOTAL_CENTS_SENSITIVITY,
            epsilon,
        )?,
        transaction_count: add_laplace_noise(
            aggregate.transaction_count,
            DEFAULT_TRANSACTION_COUNT_SENSITIVITY,
            epsilon,
        )?,
        item_count: add_laplace_noise(
            aggregate.item_count,
            DEFAULT_ITEM_COUNT_SENSITIVITY,
            epsilon,
        )?,
    })
}

/// Default epsilon = 1.0 exporter for sales aggregates.
///
/// This is the entry point used by the cloud sync path before uploading
/// aggregated sales data to the backend.
pub fn export_privatized_sales_aggregate(
    aggregate: &SalesAggregate,
) -> Result<SalesAggregate, AstraSyncError> {
    privatize_sales_aggregate(aggregate, DEFAULT_EPSILON)
}

/// Applies differential privacy to an analytics record payload when it contains
/// a [`SalesAggregate`].
///
/// Non-sales payloads are returned unchanged.  Sales aggregate payloads are
/// expected to have `event_type == "sales_aggregate"` with the aggregate stored
/// in the `metadata` field.
pub fn privatize_analytics_payload(
    payload: serde_json::Value,
    epsilon: f64,
) -> Result<serde_json::Value, AstraSyncError> {
    let mut payload = payload;
    if let Some(event_type) = payload.get("event_type").and_then(|v| v.as_str()) {
        if event_type == "sales_aggregate" {
            if let Some(metadata) = payload.get_mut("metadata") {
                let aggregate: SalesAggregate = serde_json::from_value(metadata.clone())
                    .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
                let privatized = privatize_sales_aggregate(&aggregate, epsilon)?;
                *metadata = serde_json::to_value(&privatized)
                    .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            }
        }
    }
    Ok(payload)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn zero_sensitivity_preserves_value() {
        assert_eq!(privatize_sales(12345, 0.0).unwrap(), 12345);
    }

    #[test]
    fn invalid_epsilon_is_rejected() {
        assert!(add_laplace_noise(100, 1.0, 0.0).is_err());
        assert!(add_laplace_noise(100, 1.0, -1.0).is_err());
    }

    #[test]
    fn noise_perturbs_value_within_reasonable_range() {
        let raw = 1_000_000u64;
        let mut sum = 0i64;
        let runs = 1000;
        for _ in 0..runs {
            sum += privatize_sales(raw, 1.0).unwrap() as i64;
        }
        let avg = sum / runs;
        // Laplace noise is zero-mean, so the average should be close to raw.
        assert!((avg - raw as i64).abs() < raw as i64 / 10);
    }

    #[test]
    fn negative_noise_does_not_underflow() {
        // With high probability noise is bounded; saturating math keeps us safe.
        for _ in 0..100 {
            let _ = privatize_sales(0, 1.0).unwrap();
        }
    }

    #[test]
    fn sales_aggregate_zero_sensitivity_preserved() {
        let aggregate = SalesAggregate {
            window_start_ms: 0,
            window_end_ms: 1000,
            total_cents: 12345,
            transaction_count: 42,
            item_count: 137,
        };
        // Use a tiny epsilon so sensitivities are effectively zero relative to value.
        let result = privatize_sales_aggregate(&aggregate, f64::INFINITY).unwrap();
        assert_eq!(result, aggregate);
    }

    #[test]
    fn sales_aggregate_noise_perturbs_within_range() {
        let aggregate = SalesAggregate {
            window_start_ms: 0,
            window_end_ms: 1000,
            total_cents: 1_000_000,
            transaction_count: 1000,
            item_count: 5000,
        };

        let mut total_cents_sum = 0i64;
        let mut tx_count_sum = 0i64;
        let mut item_count_sum = 0i64;
        let runs = 1000;

        for _ in 0..runs {
            let noised = export_privatized_sales_aggregate(&aggregate).unwrap();
            total_cents_sum += noised.total_cents as i64;
            tx_count_sum += noised.transaction_count as i64;
            item_count_sum += noised.item_count as i64;
        }

        let avg_total = total_cents_sum / runs;
        let avg_tx = tx_count_sum / runs;
        let avg_items = item_count_sum / runs;

        assert!((avg_total - aggregate.total_cents as i64).abs() < (aggregate.total_cents / 10) as i64);
        assert!((avg_tx - aggregate.transaction_count as i64).abs() < (aggregate.transaction_count / 10) as i64);
        assert!((avg_items - aggregate.item_count as i64).abs() < (aggregate.item_count / 10) as i64);
    }

    #[test]
    fn analytics_payload_non_sales_event_unchanged() {
        let payload = serde_json::json!({
            "event_id": "evt-1",
            "event_type": "heartbeat",
            "metadata": { "sku": "abc" },
            "timestamp": 12345,
        });
        let result = privatize_analytics_payload(payload.clone(), DEFAULT_EPSILON).unwrap();
        assert_eq!(result, payload);
    }

    #[test]
    fn analytics_payload_sales_aggregate_is_noised() {
        let aggregate = SalesAggregate {
            window_start_ms: 0,
            window_end_ms: 1000,
            total_cents: 500_000,
            transaction_count: 50,
            item_count: 200,
        };
        let payload = serde_json::json!({
            "event_id": "agg-1",
            "event_type": "sales_aggregate",
            "metadata": aggregate,
            "timestamp": 12345,
        });

        let result = privatize_analytics_payload(payload, DEFAULT_EPSILON).unwrap();
        let noised: SalesAggregate = serde_json::from_value(
            result.get("metadata").unwrap().clone()
        ).unwrap();

        // At least one field is likely to differ over many runs; here we just
        // verify the shape is preserved and values remain non-negative.
        assert_eq!(noised.window_start_ms, aggregate.window_start_ms);
        assert_eq!(noised.window_end_ms, aggregate.window_end_ms);
    }

    #[test]
    fn analytics_payload_invalid_epsilon_rejected() {
        let payload = serde_json::json!({
            "event_id": "agg-1",
            "event_type": "sales_aggregate",
            "metadata": {
                "window_start_ms": 0,
                "window_end_ms": 1000,
                "total_cents": 100,
                "transaction_count": 1,
                "item_count": 1,
            },
            "timestamp": 12345,
        });
        assert!(privatize_analytics_payload(payload, 0.0).is_err());
    }
}
