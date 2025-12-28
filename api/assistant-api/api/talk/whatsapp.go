// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/commons"
)

// WhatsappReciever handles incoming WhatsApp messages and sends a response.
// @Router /v1/whatsapp/:assistantId [post]
// @Summary Recieve whatsapp message and respond
// @Produce json
// @Success 200 {object} commons.Response
// @Failure 500 {object} commons.Response
func (cApi *ConversationApi) WhatsappReciever(c *gin.Context) {
	assistantId := c.Param("assistantId") // Extract assistantId from URL
	from := c.PostForm("From")            // Sender's WhatsApp number
	body := c.PostForm("Body")            // Message content
	// Logging the received message
	fmt.Printf("Assistant ID: %s, Received message from %s: %s\n", assistantId, from, body)

	// Responding with a simple text message
	responseMessage := "Hello! Thanks for your message. We'll get back to you shortly."

	// Call a function to send WhatsApp message using Twilio
	err := sendWhatsAppMessage(from, responseMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, commons.Response{
			Code:    400,
			Success: false,
			Data:    &commons.ErrorMessage{},
		})
		return
	}

	c.JSON(http.StatusOK, commons.Response{
		Code:    200,
		Success: true,
	})
}

// Helper function to send WhatsApp message using Twilio
func sendWhatsAppMessage(to, body string) error {
	accountSid := "YOUR_TWILIO_ACCOUNT_SID"
	authToken := "YOUR_TWILIO_AUTH_TOKEN"
	from := "whatsapp:+YOUR_TWILIO_NUMBER"

	msgData := url.Values{}
	msgData.Set("To", to)
	msgData.Set("From", from)
	msgData.Set("Body", body)
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://api.twilio.com/2010-04-01/Accounts/"+accountSid+"/Messages.json", &msgDataReader)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}

	return nil
}
