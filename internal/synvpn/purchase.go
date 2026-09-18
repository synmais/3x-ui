package synvpn

import (
	"sync"
	"time"
)

type PurchaseKind string

const (
	PurchaseCreate PurchaseKind = "create"
	PurchaseRenew  PurchaseKind = "renew"
)

type PurchaseState struct {
	TgID, UpdatedAt                int64
	Comment, ClientEmail, TariffID string
	Kind                           PurchaseKind
	Months, CarryoverDays          int
}

type PurchaseStore struct {
	mu    sync.Mutex
	items map[int64]PurchaseState
}

const purchaseTTL = time.Hour

func NewPurchaseStore() *PurchaseStore {
	return &PurchaseStore{
		items: make(map[int64]PurchaseState),
	}
}

func (s *PurchaseStore) Set(chatID int64, state PurchaseState) {
	s.mu.Lock()
	s.items[chatID] = state
	s.mu.Unlock()
}

func (s *PurchaseStore) Get(chatID int64) (PurchaseState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.items[chatID]
	if !ok || time.Since(time.UnixMilli(state.UpdatedAt)) > purchaseTTL {
		delete(s.items, chatID)
		return PurchaseState{}, false
	}

	return state, true
}

func (s *PurchaseStore) Clear(chatID int64) {
	s.mu.Lock()
	delete(s.items, chatID)
	s.mu.Unlock()
}
