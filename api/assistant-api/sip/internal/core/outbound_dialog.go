// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_outbound "github.com/rapidaai/api/assistant-api/sip/internal/outbound"
)

// outboundDialog owns SIP dialog signaling for an outbound INVITE.
// It stores the INVITE/200 OK pair so ACK, CANCEL, BYE, and route-set handling stay together.
type outboundDialog struct {
	server        *Server
	session       *Session
	request       OutboundInviteRequest
	dialogSession *sipgo.DialogClientSession
}

// NewOutboundDialog creates the SIP signaling owner for an outbound call.
func NewOutboundDialog(server *Server, session *Session, request OutboundInviteRequest) *outboundDialog {
	return &outboundDialog{
		server:  server,
		session: session,
		request: request,
	}
}

func (dialog *outboundDialog) Invite(ctx context.Context, sdpOffer string) error {
	if dialog.session == nil {
		return fmt.Errorf("outbound dialog requires a session")
	}
	if err := dialog.request.Validate(); err != nil {
		return err
	}
	callID := dialog.session.GetCallID()
	if callID == "" {
		return fmt.Errorf("%w: outbound call ID is required", ErrInvalidConfig)
	}

	recipient := sip.Uri{
		Scheme: internal_outbound.SIPScheme(internal_outbound.Transport(dialog.request.Config.Transport)),
		Host:   dialog.request.Config.Address,
		Port:   dialog.request.Config.Port,
		User:   dialog.request.Identity.ToUser,
	}
	if dialog.request.Config.Transport == TransportTLS || dialog.request.Config.Transport == TransportTCP {
		if recipient.UriParams == nil {
			recipient.UriParams = sip.NewParams()
		}
		recipient.UriParams.Add("transport", string(dialog.request.Config.Transport))
	}

	inviteHeaders, err := internal_outbound.BuildInviteHeaders(outboundDialogRequestToSignaling(dialog.request))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	callIDHeader := sip.CallIDHeader(callID)
	inviteHeaders = append(inviteHeaders, &callIDHeader)

	dialogSession, err := dialog.server.dialogClientCache.Invite(ctx, recipient, []byte(sdpOffer), inviteHeaders...)
	if err != nil {
		return fmt.Errorf("failed to send INVITE: %w", err)
	}

	dialog.dialogSession = dialogSession
	dialog.session.SetDialogClientSession(dialogSession)
	return nil
}

func (dialog *outboundDialog) WaitAnswer(ctx context.Context, options sipgo.AnswerOptions) error {
	if dialog.dialogSession == nil {
		return fmt.Errorf("outbound dialog session is required before waiting for answer")
	}
	return dialog.dialogSession.WaitAnswer(ctx, options)
}

func (dialog *outboundDialog) LogAuthChallenge(response *sip.Response, auth SIPAuthConfig) {
	if dialog == nil || dialog.server == nil || response == nil {
		return
	}
	callID := ""
	if dialog.session != nil {
		callID = dialog.session.GetCallID()
	}
	statusCode := response.StatusCode

	if statusCode == 401 {
		if wwwAuth := response.GetHeader("WWW-Authenticate"); wwwAuth != nil {
			dialog.server.logger.Debugw("SIP 401 challenge received",
				"call_id", callID,
				"www_authenticate", wwwAuth.Value(),
				"auth_username", auth.Username)
		}
		if inviteRequest := dialog.InviteRequest(); inviteRequest != nil {
			if authHeader := inviteRequest.GetHeader("Authorization"); authHeader != nil {
				dialog.server.logger.Debugw("SIP digest Authorization sent",
					"call_id", callID,
					"has_authorization", true)
			}
		}
	}
	if statusCode == 407 {
		if proxyAuth := response.GetHeader("Proxy-Authenticate"); proxyAuth != nil {
			dialog.server.logger.Debugw("SIP 407 challenge received",
				"call_id", callID,
				"proxy_authenticate", proxyAuth.Value(),
				"auth_username", auth.Username)
		}
		if inviteRequest := dialog.InviteRequest(); inviteRequest != nil {
			if authHeader := inviteRequest.GetHeader("Proxy-Authorization"); authHeader != nil {
				dialog.server.logger.Debugw("SIP digest Proxy-Authorization sent",
					"call_id", callID,
					"has_proxy_authorization", true)
			}
		}
	}
}

func (dialog *outboundDialog) AckAnswer(ctx context.Context) error {
	if dialog.dialogSession == nil || dialog.dialogSession.InviteRequest == nil || dialog.dialogSession.InviteResponse == nil {
		return fmt.Errorf("outbound answered dialog is not available")
	}
	internal_outbound.NormalizeDialogRouteSet(dialog.dialogSession)
	ackRequest := internal_outbound.NewAckRequest(dialog.dialogSession.InviteRequest, dialog.dialogSession.InviteResponse)
	return dialog.dialogSession.WriteAck(ctx, ackRequest)
}

func (dialog *outboundDialog) CancelBeforeAnswer(ctx context.Context) error {
	if dialog.dialogSession == nil || dialog.dialogSession.InviteRequest == nil {
		return nil
	}
	_, err := internal_outbound.SendCancel(ctx, dialog.dialogSession, dialog.dialogSession.InviteRequest)
	return err
}

func (dialog *outboundDialog) SendBye(ctx context.Context) error {
	if dialog.dialogSession == nil {
		return nil
	}
	if dialog.dialogSession.InviteRequest == nil || dialog.dialogSession.InviteResponse == nil {
		return dialog.dialogSession.Bye(ctx)
	}
	internal_outbound.NormalizeDialogRouteSet(dialog.dialogSession)
	byeRequest := internal_outbound.NewByeRequest(dialog.dialogSession.InviteRequest, dialog.dialogSession.InviteResponse)
	return dialog.dialogSession.WriteBye(ctx, byeRequest)
}

func (dialog *outboundDialog) Close() {
	if dialog == nil || dialog.dialogSession == nil {
		return
	}
	dialog.dialogSession.Close()
}

func (dialog *outboundDialog) CloseAfter(delay time.Duration) {
	if dialog == nil || dialog.dialogSession == nil {
		return
	}
	time.AfterFunc(delay, func() {
		dialog.Close()
	})
}

func (dialog *outboundDialog) Context() context.Context {
	if dialog == nil || dialog.dialogSession == nil {
		return nil
	}
	return dialog.dialogSession.Context()
}

func (dialog *outboundDialog) Done() <-chan struct{} {
	if dialog == nil || dialog.dialogSession == nil {
		return nil
	}
	return dialog.dialogSession.Context().Done()
}

func (dialog *outboundDialog) InviteRequest() *sip.Request {
	if dialog == nil || dialog.dialogSession == nil {
		return nil
	}
	return dialog.dialogSession.InviteRequest
}

func (dialog *outboundDialog) InviteResponse() *sip.Response {
	if dialog == nil || dialog.dialogSession == nil {
		return nil
	}
	return dialog.dialogSession.InviteResponse
}

func outboundDialogRequestToSignaling(request OutboundInviteRequest) internal_outbound.InviteRequest {
	return internal_outbound.InviteRequest{
		Config: internal_outbound.Config{
			Address:   request.Config.Address,
			Port:      request.Config.Port,
			Transport: internal_outbound.Transport(request.Config.Transport),
			Domain:    request.Config.Domain,
			Headers:   request.Config.Headers,
		},
		Identity: internal_outbound.Identity{
			ToUser:   request.Identity.ToUser,
			FromUser: request.Identity.FromUser,
		},
	}
}
