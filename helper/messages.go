package helper

import "fmt"

// Parameterized error message functions

// ErrTypeMismatch generates a type mismatch error message for a given message type
func ErrTypeMismatch(msgType string) string {
	return fmt.Sprintf("Type mismatch for %s", msgType)
}

// ErrInvalidSideTxMsg generates an error message for invalid side-tx messages
func ErrInvalidSideTxMsg(msgType string) string {
	return fmt.Sprintf("Invalid side-tx msg for %s", msgType)
}

// ErrSkippingMsg generates a message for when a handler is skipped
func ErrSkippingMsg(msgType string) string {
	return fmt.Sprintf("Skipping %s since side-tx didn't get yes votes", msgType)
}

// LogValidatingExternalCall generates a debug log message for validation
func LogValidatingExternalCall(msgType string) string {
	return fmt.Sprintf("Validating external call for %s msg", msgType)
}

// LogSuccessfullyValidated generates a success log message after validation
func LogSuccessfullyValidated(msgType string) string {
	return fmt.Sprintf("Successfully validated external call for %s msg", msgType)
}

// LogValidatingMsg generates a validation start message
func LogValidatingMsg(msgType string) string {
	return fmt.Sprintf("Validating %s msg", msgType)
}

// ErrIncorrectNonceDuringPostHandle generates a nonce error message for post-handlers
func ErrIncorrectNonceDuringPostHandle(handlerType string) string {
	return fmt.Sprintf("Incorrect validator nonce during PostHandler %s", handlerType)
}

// LogEventAlreadyProcessedIn generates the module-specific event processing error
func LogEventAlreadyProcessedIn(moduleName string) string {
	return fmt.Sprintf("Event already processed in %s sideHandler", moduleName)
}

// LogReceivedTaskToSend generates a log message for received bridge tasks
func LogReceivedTaskToSend(msgType string) string {
	return fmt.Sprintf("Received task to send %s to heimdall", msgType)
}

// ErrFailedToGetValidator generates an error message for validator retrieval failures
func ErrFailedToGetValidator(validatorAddr string) string {
	return fmt.Sprintf("Failed to get validator %s", validatorAddr)
}

// ErrFailedToGetValidatorPublicKey generates an error message for validator public key retrieval failures
func ErrFailedToGetValidatorPublicKey(validatorAddr string) string {
	return fmt.Sprintf("Failed to get validator %s public key", validatorAddr)
}
