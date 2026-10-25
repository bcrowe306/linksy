package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/service"
)

func (h *Handler) GetBilling(c fiber.Ctx) error {
	b, err := h.svc.Billing(middleware.Actor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(b)
}

type checkoutRequest struct {
	Interval string `json:"interval"`
	Code     string `json:"code"`
}

func (h *Handler) PreviewPromo(c fiber.Ctx) error {
	var req checkoutRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	p, err := h.svc.PreviewPromo(middleware.Actor(c), req.Code, req.Interval)
	if err != nil {
		if errors.Is(err, billing.ErrInvalidCode) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": "invalid_code"})
		}
		return fail(c, err)
	}
	return c.JSON(p)
}

func (h *Handler) StartCheckout(c fiber.Ctx) error {
	var req checkoutRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	url, err := h.svc.StartCheckout(middleware.Actor(c), req.Interval, req.Code)
	if err != nil {
		if errors.Is(err, billing.ErrInvalidCode) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": "invalid_code"})
		}
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"url": url})
}

func (h *Handler) BillingPortal(c fiber.Ctx) error {
	url, err := h.svc.PortalURL(middleware.Actor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"url": url})
}

// StripeWebhook receives Stripe events (signature-verified, processed once).
func (h *Handler) StripeWebhook(c fiber.Ctx) error {
	err := h.svc.HandleStripeEvent(c.Body(), c.Get("Stripe-Signature"))
	switch {
	case err == nil:
		return c.JSON(fiber.Map{"received": true})
	case errors.Is(err, billing.ErrBadSignature):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid signature"})
	case errors.Is(err, service.ErrBillingDisabled):
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "billing disabled"})
	}
	// 500 makes Stripe retry later (e.g. transient Stripe or database errors).
	log.Printf("[billing] webhook processing failed: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "processing failed"})
}
