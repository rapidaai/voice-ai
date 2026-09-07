// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

const outboundAllowHeaderValue = "INVITE, ACK, CANCEL, BYE, NOTIFY, REFER, MESSAGE, OPTIONS, INFO, SUBSCRIBE"

func buildInviteHeaders(request OutboundInviteRequest) ([]sip.Header, error) {
	fromHeader, err := buildFromHeader(request)
	if err != nil {
		return nil, err
	}

	fromDomain := request.Config.Domain
	if fromDomain == "" {
		fromDomain = request.Config.Address
	}
	scheme := sipScheme(request.Config.Transport)
	fromUser := strings.TrimSpace(request.Address.From)

	headers := []sip.Header{
		fromHeader,
		sip.NewHeader("P-Asserted-Identity", "<"+scheme+":"+fromUser+"@"+fromDomain+">"),
		sip.NewHeader("Allow", outboundAllowHeaderValue),
		sip.NewHeader("User-Agent", sipUserAgent),
	}
	headers = append(headers, sortedCustomHeaders(request.Config.Headers)...)
	return headers, nil
}

func buildFromHeader(request OutboundInviteRequest) (*sip.FromHeader, error) {
	fromDomain := request.Config.Domain
	if fromDomain == "" {
		fromDomain = request.Config.Address
	}
	fromUser := strings.TrimSpace(request.Address.From)
	if fromUser == "" {
		return nil, ErrOutboundFromUserRequired
	}

	fromHeader := &sip.FromHeader{
		DisplayName: fromUser,
		Address: sip.Uri{
			Scheme: sipScheme(request.Config.Transport),
			User:   fromUser,
			Host:   fromDomain,
		},
		Params: sip.NewParams(),
	}
	fromHeader.Params.Add("tag", sip.GenerateTagN(16))
	return fromHeader, nil
}

func buildContactHeader(config *ListenConfig) sip.ContactHeader {
	contactURI := sip.Uri{
		Scheme: sipScheme(config.Transport),
		Host:   config.GetExternalIP(),
		Port:   config.Port,
	}
	if config.Transport == TransportTCP || config.Transport == TransportTLS {
		contactURI.UriParams = sip.NewParams()
		contactURI.UriParams.Add("transport", string(config.Transport))
	}
	return sip.ContactHeader{Address: contactURI}
}

func sipScheme(transport Transport) string {
	if transport == TransportTLS {
		return "sips"
	}
	return "sip"
}

func normalizeDialogRouteSet(dialogSession *sipgo.DialogClientSession) {
	if dialogSession == nil || dialogSession.InviteRequest == nil || dialogSession.InviteResponse == nil {
		return
	}
	if len(dialogSession.InviteResponse.GetHeaders("Record-Route")) == 0 {
		return
	}
	for dialogSession.InviteRequest.RemoveHeader("Route") {
	}
}

func newAckRequest(inviteRequest *sip.Request, inviteResponse *sip.Response) *sip.Request {
	recipient := &inviteRequest.Recipient
	if contact := inviteResponse.Contact(); contact != nil {
		recipient = &contact.Address
	}

	ackRequest := sip.NewRequest(sip.ACK, *recipient.Clone())
	ackRequest.SipVersion = inviteRequest.SipVersion
	sip.CopyHeaders("Route", inviteRequest, ackRequest)
	appendDialogHeaders(ackRequest, inviteRequest, inviteResponse, sip.ACK)

	if contact := inviteRequest.Contact(); contact != nil {
		ackRequest.AppendHeader(sip.HeaderClone(contact))
	}
	ackRequest.AppendHeader(sip.NewHeader("User-Agent", sipUserAgent))
	ackRequest.SetTransport(inviteRequest.Transport())
	ackRequest.SetSource(inviteRequest.Source())
	ackRequest.Laddr = inviteRequest.Laddr
	return ackRequest
}

func newByeRequest(inviteRequest *sip.Request, inviteResponse *sip.Response) *sip.Request {
	recipient := &inviteRequest.Recipient
	if contact := inviteResponse.Contact(); contact != nil {
		recipient = &contact.Address
	}

	byeRequest := sip.NewRequest(sip.BYE, *recipient.Clone())
	byeRequest.SipVersion = inviteRequest.SipVersion
	sip.CopyHeaders("Route", inviteRequest, byeRequest)
	appendDialogHeaders(byeRequest, inviteRequest, inviteResponse, sip.BYE)
	byeRequest.AppendHeader(sip.NewHeader("User-Agent", sipUserAgent))
	byeRequest.SetTransport(inviteRequest.Transport())
	byeRequest.SetSource(inviteRequest.Source())
	byeRequest.Laddr = inviteRequest.Laddr
	return byeRequest
}

func newCancelRequest(inviteRequest *sip.Request) *sip.Request {
	cancelRequest := sip.NewRequest(sip.CANCEL, inviteRequest.Recipient)
	cancelRequest.SipVersion = inviteRequest.SipVersion
	if via := inviteRequest.Via(); via != nil {
		cancelRequest.AppendHeader(sip.HeaderClone(via))
	}
	maxForwardsHeader := sip.MaxForwardsHeader(70)
	cancelRequest.AppendHeader(&maxForwardsHeader)
	if from := inviteRequest.From(); from != nil {
		cancelRequest.AppendHeader(sip.HeaderClone(from))
	}
	if to := inviteRequest.To(); to != nil {
		cancelRequest.AppendHeader(sip.HeaderClone(to))
	}
	if callID := inviteRequest.CallID(); callID != nil {
		cancelRequest.AppendHeader(sip.HeaderClone(callID))
	}
	if cseq := inviteRequest.CSeq(); cseq != nil {
		cancelRequest.AppendHeader(sip.HeaderClone(cseq))
		if cancelCSeq := cancelRequest.CSeq(); cancelCSeq != nil {
			cancelCSeq.MethodName = sip.CANCEL
		}
	}
	sip.CopyHeaders("Route", inviteRequest, cancelRequest)
	cancelRequest.AppendHeader(sip.NewHeader("User-Agent", sipUserAgent))
	cancelRequest.SetTransport(inviteRequest.Transport())
	cancelRequest.SetSource(inviteRequest.Source())
	cancelRequest.Laddr = inviteRequest.Laddr
	return cancelRequest
}

func sendCancel(ctx context.Context, dialogSession *sipgo.DialogClientSession, inviteRequest *sip.Request) (*sip.Response, error) {
	if dialogSession == nil || dialogSession.UA == nil || dialogSession.UA.Client == nil || inviteRequest == nil {
		return nil, fmt.Errorf("outbound invite dialog is not available")
	}
	cancelRequest := newCancelRequest(inviteRequest)
	return dialogSession.UA.Client.Do(ctx, cancelRequest, func(client *sipgo.Client, request *sip.Request) error {
		return nil
	})
}

func sortedCustomHeaders(headers map[string]string) []sip.Header {
	keys := make([]string, 0, len(headers))
	for name := range headers {
		if !isGeneratedHeader(name) {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)

	out := make([]sip.Header, 0, len(keys))
	for _, name := range keys {
		out = append(out, sip.NewHeader(name, headers[name]))
	}
	return out
}

func isGeneratedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "allow",
		"call-id",
		"contact",
		"content-length",
		"content-type",
		"cseq",
		"from",
		"max-forwards",
		"p-asserted-identity",
		"to",
		"user-agent",
		"via":
		return true
	default:
		return false
	}
}

func appendDialogHeaders(req *sip.Request, inviteRequest *sip.Request, inviteResponse *sip.Response, method sip.RequestMethod) {
	maxForwardsHeader := sip.MaxForwardsHeader(70)
	req.AppendHeader(&maxForwardsHeader)
	if from := inviteRequest.From(); from != nil {
		req.AppendHeader(sip.HeaderClone(from))
	}
	if to := inviteResponse.To(); to != nil {
		req.AppendHeader(sip.HeaderClone(to))
	}
	if callID := inviteRequest.CallID(); callID != nil {
		req.AppendHeader(sip.HeaderClone(callID))
	}
	if cseq := inviteRequest.CSeq(); cseq != nil {
		req.AppendHeader(sip.HeaderClone(cseq))
		if copiedCSeq := req.CSeq(); copiedCSeq != nil {
			copiedCSeq.MethodName = method
		}
	}
}
