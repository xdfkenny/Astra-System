#ifndef VERIFONE_POS_H
#define VERIFONE_POS_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Verifone SDK status codes.
 *
 * Negative values are errors; VX_OK (0) indicates success.
 */
typedef int VxStatus;

#define VX_OK 0
#define VX_ERR_NOT_INITIALIZED -1
#define VX_ERR_INVALID_PARAM -2
#define VX_ERR_TIMEOUT -3
#define VX_ERR_CARD_READ_FAILED -4
#define VX_ERR_PROCESSING -5
#define VX_ERR_NETWORK -6
#define VX_ERR_CANCELED -7
#define VX_ERR_CLOSED -8

/** ISO-4217 currency code length including the null terminator. */
#define VX_CURRENCY_LEN 4

/** Maximum length of a transaction identifier buffer including terminator. */
#define VX_TXN_ID_LEN 64

/**
 * Initialise the Verifone terminal driver.
 *
 * @param config_path Path to a driver configuration file. May be NULL for
 *                    default configuration.
 * @return VX_OK on success, or an error code.
 */
int VxInitTerminal(const char* config_path);

/**
 * Start a new payment transaction.
 *
 * @param amount_cents Transaction amount in the smallest currency unit
 *                     (e.g. cents).
 * @param currency     Three-letter ISO-4217 currency code (e.g. "USD").
 * @return VX_OK on success, or an error code.
 */
int VxStartTransaction(int64_t amount_cents, const char currency[4]);

/**
 * Wait for the cardholder to present a card.
 *
 * @param timeout_ms Maximum time to wait in milliseconds.
 * @return VX_OK when a card is presented, VX_ERR_TIMEOUT, or another error.
 */
int VxWaitForCard(uint32_t timeout_ms);

/**
 * Process the payment after a card has been presented.
 *
 * @param transaction_id     Buffer to receive the null-terminated transaction
 *                           identifier.
 * @param transaction_id_len Length of the buffer in bytes.
 * @return VX_OK on success, or an error code.
 */
int VxProcessPayment(char* transaction_id, size_t transaction_id_len);

/**
 * Refund a previous transaction.
 *
 * @param transaction_id Null-terminated transaction identifier.
 * @return VX_OK on success, or an error code.
 */
int VxRefund(const char* transaction_id);

/**
 * Close the terminal driver and release resources.
 *
 * @return VX_OK on success, or an error code.
 */
int VxCloseTerminal(void);

#ifdef __cplusplus
}
#endif

#endif /* VERIFONE_POS_H */
