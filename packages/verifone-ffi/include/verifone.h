/**
 * @file verifone.h
 * @brief C ABI for the Verifone payment terminal SDK.
 *
 * This header defines the contract between the vendor's proprietary C SDK and
 * the Rust `astra-syncd` daemon.  It is intended to be processed by bindgen
 * and wrapped by a thin, safe Rust FFI module.
 *
 * All monetary amounts are passed as the smallest currency unit (e.g. cents)
 * via unsigned 64-bit integers.  Currency codes follow ISO-4217 and are passed
 * as NUL-terminated strings.
 */

#ifndef VERIFONE_H
#define VERIFONE_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Opaque handle to a Verifone terminal session.
 *
 * Returned by vx_init_terminal and must be closed with vx_close_terminal.
 */
typedef struct VxTerminal VxTerminal;

/**
 * Result codes returned by the Verifone SDK.
 *
 * These values are stable and documented in the vendor integration guide.
 * The Rust wrapper maps every variant to a strongly-typed enum.
 */
typedef enum {
    VX_OK = 0,
    VX_ERROR_UNKNOWN = 1,
    VX_ERROR_INVALID_PARAMETER = 2,
    VX_ERROR_NOT_INITIALIZED = 3,
    VX_ERROR_ALREADY_INITIALIZED = 4,
    VX_ERROR_COMMUNICATION = 5,
    VX_ERROR_TIMEOUT = 6,
    VX_ERROR_CARD_READ = 7,
    VX_ERROR_TRANSACTION_CANCELLED = 8,
    VX_ERROR_DECLINED = 9,
    VX_ERROR_NETWORK = 10,
    VX_ERROR_REFUND_NOT_ALLOWED = 11,
    VX_ERROR_AMOUNT_EXCEEDS_LIMIT = 12,
    VX_ERROR_CURRENCY_NOT_SUPPORTED = 13,
    VX_ERROR_TERMINAL_BUSY = 14,
} VxResult;

/**
 * Card presentation result.
 */
typedef struct {
    uint8_t present;
    char brand[32];
    char last_four[5];
} VxCardInfo;

/**
 * Payment authorization result.
 */
typedef struct {
    VxResult result;
    char auth_code[16];
    char opaque_token[256];
    char terminal_id[64];
    uint64_t authorized_amount_cents;
} VxPaymentResult;

/**
 * Refund result.
 */
typedef struct {
    VxResult result;
    char reference[128];
    uint64_t refunded_amount_cents;
} VxRefundResult;

/**
 * Initializes the connection to the Verifone terminal.
 *
 * @param out_terminal pointer to receive the terminal handle.
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_init_terminal(VxTerminal** out_terminal);

/**
 * Begins a new transaction for the requested amount.
 *
 * @param terminal handle returned by vx_init_terminal.
 * @param amount_cents amount in the smallest currency unit.
 * @param currency ISO-4217 currency code (e.g. "USD").
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_start_transaction(VxTerminal* terminal,
                              uint64_t amount_cents,
                              const char* currency);

/**
 * Blocks until a card is presented or a timeout occurs.
 *
 * @param terminal handle returned by vx_init_terminal.
 * @param timeout_ms maximum time to wait in milliseconds.
 * @param out_card pointer to receive card information.
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_wait_for_card(VxTerminal* terminal,
                          uint32_t timeout_ms,
                          VxCardInfo* out_card);

/**
 * Processes the payment after a card has been presented.
 *
 * @param terminal handle returned by vx_init_terminal.
 * @param out_result pointer to receive the payment result.
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_process_payment(VxTerminal* terminal,
                            VxPaymentResult* out_result);

/**
 * Refunds a previous transaction.
 *
 * @param terminal handle returned by vx_init_terminal.
 * @param transaction_id NUL-terminated transaction identifier.
 * @param amount_cents amount to refund in the smallest currency unit.
 * @param currency ISO-4217 currency code.
 * @param out_result pointer to receive the refund result.
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_refund(VxTerminal* terminal,
                   const char* transaction_id,
                   uint64_t amount_cents,
                   const char* currency,
                   VxRefundResult* out_result);

/**
 * Closes the terminal connection and releases resources.
 *
 * @param terminal handle returned by vx_init_terminal.
 * @return VX_OK on success, otherwise an error code.
 */
VxResult vx_close_terminal(VxTerminal* terminal);

#ifdef __cplusplus
}
#endif

#endif /* VERIFONE_H */
