package main_test

// import (
// 	"fmt"
// 	"log"
// 	"math"
// 	"path/filepath"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/sugarme/tokenizer"
// 	"github.com/sugarme/tokenizer/pretrained"
// 	ort "github.com/yalue/onnxruntime_go"
// )

// // Constants
// const (
// 	UnlikelyThreshold = 0.15
// 	MaxHistoryTokens  = 1024
// )

// // ChatMessage represents a message in a conversation
// type ChatMessage struct {
// 	Role    string
// 	Content string
// }

// // ModelData holds the model components
// type ModelData struct {
// 	Tokenizer *tokenizer.Tokenizer
// 	// Session   *ort.Session
// 	EouIndex int
// }

// func TestAnotherTest(t *testing.T) {
// 	fmt.Printf("this")
// }
// func TestTrunDetector(m *testing.T) {
// 	// Set up ONNX Runtime
// 	ort.SetSharedLibraryPath("/opt/homebrew/lib/libonnxruntime.dylib") // Update with your actual path

// 	err := ort.InitializeEnvironment()
// 	if err != nil {
// 		log.Fatalf("Failed to initialize ONNX environment: %v", err)
// 	}
// 	defer ort.DestroyEnvironment()

// 	// Initialize the model
// 	modelData, err := initializeModel()
// 	if err != nil {
// 		log.Fatalf("Failed to initialize model: %v", err)
// 	}

// 	startTime := time.Now()

// 	// Example chat contexts
// 	chatExample1 := []ChatMessage{
// 		{Role: "user", Content: "What's the weather like today?"},
// 		{Role: "assistant", Content: "It's sunny and warm."},
// 		{Role: "user", Content: "I like the weather. but"},
// 		{Role: "user", Content: "I'm not sure what to"},
// 	}

// 	chatExample2 := []ChatMessage{
// 		{Role: "user", Content: "आज मौसम कैसा है?"},
// 		{Role: "assistant", Content: "आज धूप है और गर्मी है।"},
// 		{Role: "user", Content: "मुझे मौसम अच्छा लगता है, लेकिन"},
// 		{Role: "user", Content: "मुझे यह नहीं पता कि क्या"},
// 	}

// 	chatExample3 := []ChatMessage{
// 		{Role: "user", Content: "What's the weather like today?"},
// 		{Role: "assistant", Content: "It's sunny and warm."},
// 		{Role: "user", Content: "I like the weather. but"},
// 		{Role: "user", Content: "I'm not sure what to do? But maybe"},
// 	}

// 	// Run predictions
// 	examples := [][]ChatMessage{chatExample1, chatExample2, chatExample3}
// 	for i, example := range examples {
// 		probability, err := predictEndOfTurn(example, modelData)
// 		if err != nil {
// 			log.Printf("Error predicting for example %d: %v", i+1, err)
// 			continue
// 		}
// 		fmt.Printf("End of turn probability%d: %f\n", i+1, probability)
// 	}

// 	fmt.Printf("If probability is less than %f, the model predicts that the user hasn't finished speaking.\n", UnlikelyThreshold)
// 	fmt.Printf("Prediction time: %.2f seconds\n", time.Since(startTime).Seconds())
// }

// // initializeModel loads the ONNX model and tokenizer
// func initializeModel() (*ModelData, error) {
// 	startTime := time.Now()

// 	tok, err := pretrained.FromFile("models")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load tokenizer: %v", err)
// 	}

// 	eouToken := "<|im_end|>"
// 	eouIndex, err := getTokenID(tok, eouToken)
// 	// Since we'll create session during prediction, we return nil for session here
// 	fmt.Printf("Model initialization took: %.2f seconds\n", time.Since(startTime).Seconds())
// 	return &ModelData{
// 		Tokenizer: tok,
// 		// Session:   nil, // We'll create session during prediction
// 		EouIndex: eouIndex,
// 	}, nil
// }
// func getTokenID(tok *tokenizer.Tokenizer, token string) (int, error) {
// 	// Encode the token to get its ID

// 	encoding, err := tok.Encode(tokenizer.NewSingleEncodeInput(tokenizer.NewInputSequence(token)), true)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to encode token '%s': %v", token, err)
// 	}

// 	// Check if we got any IDs
// 	ids := encoding.Ids
// 	if len(ids) == 0 {
// 		return 0, fmt.Errorf("token '%s' was not found in the vocabulary", token)
// 	}

// 	// For special tokens like <|im_end|>, we usually want the first/only ID
// 	// For tokens that get split into multiple pieces, you might need to adjust this logic
// 	return int(ids[0]), nil
// }

// // getEouTokenID finds the ID for the end-of-utterance token
// func getEouTokenID(tok *tokenizer.Tokenizer) (int, error) {
// 	// Encode the EOU token
// 	encoding, err := tok.Encode("<|im_end|>", false)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if len(encoding.Ids) == 0 {
// 		return 0, fmt.Errorf("failed to encode EOU token")
// 	}
// 	return int(encoding.Ids[0]), nil
// }

// // normalize removes punctuation and standardizes whitespace
// func normalize(text string) string {
// 	// Remove punctuation except single quotes
// 	puncs := "!\"#$%&()*+,-./:;<=>?@[\\]^_`{|}~"
// 	var builder strings.Builder
// 	for _, ch := range text {
// 		if !strings.ContainsRune(puncs, ch) {
// 			builder.WriteRune(ch)
// 		}
// 	}
// 	stripped := builder.String()

// 	// Normalize whitespace and convert to lowercase
// 	words := strings.Fields(strings.ToLower(stripped))
// 	return strings.Join(words, " ")
// }

// // formatChatContext formats the chat context for model input
// func formatChatContext(chatContext []ChatMessage, tok *tokenizer.Tokenizer) string {
// 	// Normalize and filter empty messages
// 	var normalizedContext []ChatMessage
// 	for _, msg := range chatContext {
// 		normalizedContent := normalize(msg.Content)
// 		if normalizedContent != "" {
// 			normalizedContext = append(normalizedContext, ChatMessage{
// 				Role:    msg.Role,
// 				Content: normalizedContent,
// 			})
// 		}
// 	}

// 	// Apply chat template
// 	convoText := applyChatTemplate(normalizedContext)

// 	// Handle end of utterance token
// 	eouToken := "<|im_end|>"
// 	lastEouIndex := strings.LastIndex(convoText, eouToken)
// 	if lastEouIndex >= 0 {
// 		return convoText[:lastEouIndex]
// 	}
// 	return convoText
// }

// // applyChatTemplate applies a chat template similar to transformers' apply_chat_template
// func applyChatTemplate(messages []ChatMessage) string {
// 	var result strings.Builder

// 	// Implement a simplified version of the chat template
// 	for _, msg := range messages {
// 		switch msg.Role {
// 		case "user":
// 			result.WriteString("<|im_start|>user\n")
// 			result.WriteString(msg.Content)
// 			result.WriteString("<|im_end|>\n")
// 		case "assistant":
// 			result.WriteString("<|im_start|>assistant\n")
// 			result.WriteString(msg.Content)
// 			result.WriteString("<|im_end|>\n")
// 		}
// 	}

// 	// Add generation prompt for assistant
// 	result.WriteString("<|im_start|>assistant\n")

// 	return result.String()
// }

// // softmax computes softmax probabilities for logits
// func softmax(logits []float32) []float32 {
// 	// Find max for numerical stability
// 	var max float32 = -math.MaxFloat32
// 	for _, v := range logits {
// 		if v > max {
// 			max = v
// 		}
// 	}

// 	// Compute exp(logits - max)
// 	expLogits := make([]float32, len(logits))
// 	var sum float32
// 	for i, v := range logits {
// 		expLogits[i] = float32(math.Exp(float64(v - max)))
// 		sum += expLogits[i]
// 	}

// 	// Normalize
// 	result := make([]float32, len(logits))
// 	for i, v := range expLogits {
// 		result[i] = v / sum
// 	}
// 	return result
// }

// // predictEndOfTurn predicts whether the current turn is complete
// func predictEndOfTurn(chatContext []ChatMessage, modelData *ModelData) (float32, error) {
// 	tok := modelData.Tokenizer
// 	// eouIndex := modelData.EouIndex

// 	formattedText := formatChatContext(chatContext, tok)

// 	// Tokenize the input
// 	encoding, err := tok.Encode(formattedText, false)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to tokenize input: %v", err)
// 	}

// 	// Truncate if necessary
// 	inputIds := encoding.Ids
// 	if len(inputIds) > MaxHistoryTokens {
// 		inputIds = inputIds[len(inputIds)-MaxHistoryTokens:]
// 	}

// 	// Convert to int32 for onnxruntime
// 	inputIdsInt32 := make([]int32, len(inputIds))
// 	for i, id := range inputIds {
// 		inputIdsInt32[i] = int32(id)
// 	}

// 	// Create input shape for tensor
// 	inputShape := ort.NewShape(1, int64(len(inputIdsInt32)))

// 	// Create input tensor
// 	inputTensor, err := ort.NewTensor(inputShape, inputIdsInt32)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to create input tensor: %v", err)
// 	}
// 	defer inputTensor.Destroy()

// 	// Get the model's vocabulary size to create output tensor shape
// 	// This is an estimate - adjust based on your model's actual vocabulary size
// 	vocabSize := int64(50257) // Default for GPT-2, adjust for your model

// 	// Create output shape for tensor
// 	outputShape := ort.NewShape(1, int64(len(inputIdsInt32)), vocabSize)

// 	// Create output tensor
// 	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to create output tensor: %v", err)
// 	}
// 	defer outputTensor.Destroy()

// 	// Create session
// 	modelPath := filepath.Join("models", "model_quantized.onnx")
// 	session, err := ort.NewAdvancedSession(modelPath,
// 		[]string{"input_ids"},
// 		[]string{"logits"},
// 		[]ort.Value{inputTensor},
// 		[]ort.Value{outputTensor},
// 		nil)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to create session: %v", err)
// 	}
// 	defer session.Destroy()

// 	// Run inference
// 	err = session.Run()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to run inference: %v", err)
// 	}

// 	// Get output data
// 	logits := outputTensor.GetData()

// 	// Extract the last token logits
// 	seqLen := int64(len(inputIdsInt32))
// 	offset := (seqLen - 1) * vocabSize // Get the offset for the last token
// 	lastTokenLogits := make([]float32, vocabSize)

// 	// Copy the last token logits
// 	for i := int64(0); i < vocabSize; i++ {
// 		lastTokenLogits[i] = logits[offset+i]
// 	}

// 	// Apply softmax to get probabilities
// 	probs := softmax(lastTokenLogits)

// 	// Return the probability of EOU token
// 	return probs[modelData.EouIndex], nil
// }

// // The line `tok, err := pretrained.FromFile("models")` is loading a pretrained tokenizer from a file
// // located in the "models" directory. The `pretrained.FromFile` function is used to load a tokenizer
// // that has been previously saved to a file. The tokenizer is then stored in the variable `tok` for
// // later use, and any error that occurs during the loading process is stored in the `err` variable.
