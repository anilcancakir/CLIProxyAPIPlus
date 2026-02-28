// Package openai provides HTTP handlers for OpenAI API endpoints.
// This file implements the OpenAI Images API for image generation.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers"
	"github.com/tidwall/gjson"
)

// OpenAIImageFormat represents the OpenAI Images API format identifier.
const OpenAIImageFormat = "openai-images"

// ImageGenerationRequest represents the OpenAI image generation request format.
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	User           string `json:"user,omitempty"`
}

// ImageGenerationResponse represents the OpenAI image generation response format.
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents a single generated image.
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// OpenAIImagesAPIHandler contains the handlers for OpenAI Images API endpoints.
type OpenAIImagesAPIHandler struct {
	*handlers.BaseAPIHandler
}

// NewOpenAIImagesAPIHandler creates a new OpenAI Images API handlers instance.
//
// Parameters:
//   - apiHandlers: The base API handlers instance
//
// Returns:
//   - *OpenAIImagesAPIHandler: A new OpenAI Images API handlers instance
func NewOpenAIImagesAPIHandler(apiHandlers *handlers.BaseAPIHandler) *OpenAIImagesAPIHandler {
	return &OpenAIImagesAPIHandler{
		BaseAPIHandler: apiHandlers,
	}
}

// HandlerType returns the identifier for this handler implementation.
func (h *OpenAIImagesAPIHandler) HandlerType() string {
	return OpenAIImageFormat
}

// Models returns the image-capable models supported by this handler.
func (h *OpenAIImagesAPIHandler) Models() []map[string]any {
	modelRegistry := registry.GetGlobalRegistry()
	return modelRegistry.GetAvailableModels("openai")
}

// ImageGenerations handles the /v1/images/generations endpoint.
// It supports OpenAI DALL-E and Gemini Imagen models through a unified interface.
//
// Request format (OpenAI-compatible):
//
//	{
//	  "model": "dall-e-3" | "imagen-4.0-generate-001" | "gemini-2.5-flash-image",
//	  "prompt": "A white siamese cat",
//	  "n": 1,
//	  "quality": "standard" | "hd",
//	  "response_format": "url" | "b64_json",
//	  "size": "1024x1024" | "1024x1792" | "1792x1024",
//	  "style": "vivid" | "natural"
//	}
//
// Response format:
//
//	{
//	  "created": 1589478378,
//	  "data": [
//	    {
//	      "url": "https://..." | "b64_json": "base64...",
//	      "revised_prompt": "..."
//	    }
//	  ]
//	}
func (h *OpenAIImagesAPIHandler) ImageGenerations(c *gin.Context) {
	rawJSON, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	modelName := gjson.GetBytes(rawJSON, "model").String()
	if modelName == "" {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: "model is required",
				Type:    "invalid_request_error",
				Code:    "missing_model",
			},
		})
		return
	}

	prompt := gjson.GetBytes(rawJSON, "prompt").String()
	if prompt == "" {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: "prompt is required",
				Type:    "invalid_request_error",
				Code:    "missing_prompt",
			},
		})
		return
	}

	// 1. Convert OpenAI Images request to provider-specific format.
	providerPayload := h.convertToProviderFormat(modelName, rawJSON)

	// 2. Determine the handler type based on the model name.
	handlerType := h.determineHandlerType(modelName)

	// 3. Execute the request and write the response.
	c.Header("Content-Type", "application/json")
	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
	resp, upstreamHeaders, errMsg := h.ExecuteWithAuthManager(cliCtx, handlerType, modelName, providerPayload, h.GetAlt(c))
	if errMsg != nil {
		h.WriteErrorResponse(c, errMsg)
		if errMsg.Error != nil {
			cliCancel(errMsg.Error)
		} else {
			cliCancel(nil)
		}
		return
	}
	handlers.WriteUpstreamHeaders(c.Writer.Header(), upstreamHeaders)

	// 4. Convert provider response to OpenAI Images format.
	responseFormat := gjson.GetBytes(rawJSON, "response_format").String()
	openAIResponse := h.convertToOpenAIFormat(resp, modelName, prompt, responseFormat)

	c.JSON(http.StatusOK, openAIResponse)
	cliCancel()
}

// convertToProviderFormat converts OpenAI Images API request to provider-specific format.
func (h *OpenAIImagesAPIHandler) convertToProviderFormat(modelName string, rawJSON []byte) []byte {
	// Check if this is a Gemini/Imagen model — requires contents-style payload.
	if h.isGeminiImageModel(modelName) {
		return h.convertToGeminiFormat(rawJSON)
	}

	// For OpenAI DALL-E and other models, pass through with minimal transformation.
	// The OpenAI compatibility executor handles the rest.
	return rawJSON
}

// convertToGeminiFormat converts OpenAI Images request to Gemini format.
func (h *OpenAIImagesAPIHandler) convertToGeminiFormat(rawJSON []byte) []byte {
	prompt := gjson.GetBytes(rawJSON, "prompt").String()
	model := gjson.GetBytes(rawJSON, "model").String()
	n := gjson.GetBytes(rawJSON, "n").Int()
	size := gjson.GetBytes(rawJSON, "size").String()

	// Build Gemini-style request using the contents format that Gemini executors understand.
	geminiReq := map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]any{{"text": prompt}},
			},
		},
		"generationConfig": map[string]any{
			"responseModalities": []string{"IMAGE", "TEXT"},
		},
	}

	// Map OpenAI size to Gemini aspect ratio.
	if size != "" {
		aspectRatio := h.mapSizeToAspectRatio(size)
		if aspectRatio != "" {
			geminiReq["generationConfig"].(map[string]any)["imageConfig"] = map[string]any{
				"aspectRatio": aspectRatio,
			}
		}
	}

	// Handle n (number of images) — Gemini uses sampleCount.
	if n > 1 {
		geminiReq["generationConfig"].(map[string]any)["sampleCount"] = int(n)
	}

	// Set model field if provided.
	if model != "" {
		geminiReq["model"] = model
	}

	result, err := json.Marshal(geminiReq)
	if err != nil {
		return rawJSON
	}
	return result
}

// mapSizeToAspectRatio maps OpenAI image sizes to Gemini aspect ratios.
func (h *OpenAIImagesAPIHandler) mapSizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
	case "1792x1024":
		return "16:9"
	case "1024x1792":
		return "9:16"
	case "512x512":
		return "1:1"
	case "256x256":
		return "1:1"
	default:
		return "1:1"
	}
}

// isGeminiImageModel checks if the model is a Gemini or Imagen image model.
func (h *OpenAIImagesAPIHandler) isGeminiImageModel(model string) bool {
	return imageModelContains(model, "imagen") ||
		imageModelContains(model, "gemini-2.5-flash-image") ||
		imageModelContains(model, "gemini-3-pro-image")
}

// determineHandlerType determines the handler type based on the model name.
func (h *OpenAIImagesAPIHandler) determineHandlerType(modelName string) string {
	// Gemini/Imagen models use the Gemini executor.
	if h.isGeminiImageModel(modelName) {
		return constant.Gemini
	}

	// Default to OpenAI for DALL-E and other models.
	return constant.OpenAI
}

// convertToOpenAIFormat converts provider response to OpenAI Images API response format.
func (h *OpenAIImagesAPIHandler) convertToOpenAIFormat(
	resp []byte,
	modelName string,
	originalPrompt string,
	responseFormat string,
) *ImageGenerationResponse {
	created := time.Now().Unix()

	// Check if this is a Gemini-style response.
	if h.isGeminiImageModel(modelName) {
		return h.convertGeminiToOpenAI(resp, created, originalPrompt, responseFormat)
	}

	// Try to parse as OpenAI-style response directly.
	var openAIResp ImageGenerationResponse
	if err := json.Unmarshal(resp, &openAIResp); err == nil && len(openAIResp.Data) > 0 {
		return &openAIResp
	}

	// Fallback: wrap raw response as b64_json.
	return &ImageGenerationResponse{
		Created: created,
		Data: []ImageData{
			{
				B64JSON:       string(resp),
				RevisedPrompt: originalPrompt,
			},
		},
	}
}

// convertGeminiToOpenAI converts Gemini image response to OpenAI Images format.
func (h *OpenAIImagesAPIHandler) convertGeminiToOpenAI(
	resp []byte,
	created int64,
	originalPrompt string,
	responseFormat string,
) *ImageGenerationResponse {
	response := &ImageGenerationResponse{
		Created: created,
		Data:    []ImageData{},
	}

	// Parse Gemini response — try candidates[].content.parts[] format.
	parts := gjson.GetBytes(resp, "candidates.0.content.parts")
	if parts.Exists() && parts.IsArray() {
		for _, part := range parts.Array() {
			// Check for inlineData (base64 image).
			inlineData := part.Get("inlineData")
			if inlineData.Exists() {
				data := inlineData.Get("data").String()
				mimeType := inlineData.Get("mimeType").String()

				if data != "" {
					image := ImageData{
						RevisedPrompt: originalPrompt,
					}
					if responseFormat == "b64_json" {
						image.B64JSON = data
					} else {
						image.URL = fmt.Sprintf("data:%s;base64,%s", mimeType, data)
					}
					response.Data = append(response.Data, image)
				}
			}
		}
	}

	// If no images found, return empty placeholder to avoid nil data slice.
	if len(response.Data) == 0 {
		response.Data = append(response.Data, ImageData{
			RevisedPrompt: originalPrompt,
		})
	}

	return response
}

// WriteErrorResponse writes an error message to the response writer.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//   - msg: The error message to write
func (h *OpenAIImagesAPIHandler) WriteErrorResponse(c *gin.Context, msg *interfaces.ErrorMessage) {
	status := http.StatusInternalServerError
	if msg != nil && msg.StatusCode > 0 {
		status = msg.StatusCode
	}

	errText := http.StatusText(status)
	if msg != nil && msg.Error != nil {
		if v := msg.Error.Error(); v != "" {
			errText = v
		}
	}

	body := handlers.BuildErrorResponseBody(status, errText)

	if !c.Writer.Written() {
		c.Writer.Header().Set("Content-Type", "application/json")
	}
	c.Status(status)
	_, _ = c.Writer.Write(body)
}

// imageModelContains checks if s contains substr using a simple case-insensitive scan.
func imageModelContains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	return s == substr || (len(s) > len(substr) && imageModelContainsSubstring(s, substr))
}

// imageModelContainsSubstring performs a case-insensitive substring scan.
func imageModelContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
