package ledger

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestLedger_BalanceEqualsSumOfEntries(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_LEDGER_TEST_1"
	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 0,
	})

	// 1. Dolum: +100 TL (10000 kuruş)
	_, err := svc.Recharge(ctx, cardID, 10000, "REF_DOLUM_1", "Kiosk dolumu")
	if err != nil {
		t.Fatalf("Dolum başarısız: %v", err)
	}

	// 2. Biniş: -20 TL (2000 kuruş)
	_, err = svc.ProcessTap(ctx, cardID, 2000, "VAL_TAP_1", "Otobüs binişi", 0)
	if err != nil {
		t.Fatalf("Biniş başarısız: %v", err)
	}

	// 3. Aktarma Binişi: -10 TL (1000 kuruş)
	_, err = svc.ProcessTap(ctx, cardID, 1000, "VAL_TAP_2", "Tramvay aktarması", 0)
	if err != nil {
		t.Fatalf("Aktarma binişi başarısız: %v", err)
	}

	// 4. İade: +5 TL (500 kuruş)
	_, err = svc.Refund(ctx, cardID, 500, "REF_IADE_1", "Gecikme iadesi")
	if err != nil {
		t.Fatalf("İade başarısız: %v", err)
	}

	// 5. Storno Düzeltme: +10 TL (1000 kuruş)
	_, err = svc.StornoCorrection(ctx, cardID, 1000, "REF_STORNO_1", "Hatalı çekim düzeltmesi")
	if err != nil {
		t.Fatalf("Storno başarısız: %v", err)
	}

	// Bakiye doğrulaması: 10000 - 2000 - 1000 + 500 + 1000 = 8500 kuruş (85.00 TL)
	currentBalance, err := svc.GetCardBalance(ctx, cardID)
	if err != nil {
		t.Fatalf("Bakiye okuma hatası: %v", err)
	}
	if currentBalance != 8500 {
		t.Fatalf("Beklenen güncel bakiye 8500 kuruş, bulunan: %d", currentBalance)
	}

	// Tüm defter hareketlerini topla: SUM(amount_kurus) == currentBalance olmalı!
	entries, err := svc.GetCardLedger(ctx, cardID)
	if err != nil {
		t.Fatalf("Defter okuma hatası: %v", err)
	}

	var sumKurus domain.Kurus
	for _, e := range entries {
		sumKurus += e.AmountKurus
	}

	if sumKurus != currentBalance {
		t.Fatalf("Defter hareketleri toplamı (%d kuruş) ile kart bakiyesi (%d kuruş) uyuşmuyor!",
			sumKurus, currentBalance)
	}
}

func TestLedger_ConcurrentTapsRaceCondition(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_CONCURRENT_RACE"
	initialBalance := domain.Kurus(250000) // 2500.00 TL

	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	numGoroutines := 100
	farePerTap := domain.Kurus(2000) // Her okumada 20.00 TL (Toplam 2000 TL harcanacak)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			refID := fmt.Sprintf("CONCURRENT_VAL_%d", idx)
			_, err := svc.ProcessTap(ctx, cardID, farePerTap, refID, "Eşzamanlı okuma", 0)
			if err != nil {
				t.Errorf("Eşzamanlı binişte hata: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// 1000 TL - (100 * 20 TL) = -1000 TL = 0 TL (100000 - 200000)
	expectedFinalBalance := initialBalance - (domain.Kurus(numGoroutines) * farePerTap)
	finalBalance, err := svc.GetCardBalance(ctx, cardID)
	if err != nil {
		t.Fatalf("Bakiye okuma hatası: %v", err)
	}

	if finalBalance != expectedFinalBalance {
		t.Fatalf("Eşzamanlılık yarış durumu (Race Condition) tespit edildi! Beklenen: %d kuruş, Bulunan: %d kuruş",
			expectedFinalBalance, finalBalance)
	}

	// Defter toplamını da kontrol et
	entries, err := svc.GetCardLedger(ctx, cardID)
	if err != nil {
		t.Fatalf("Defter okuma hatası: %v", err)
	}
	if len(entries) != numGoroutines {
		t.Fatalf("Defterdeki hareket sayısı eksik! Beklenen: %d, Bulunan: %d", numGoroutines, len(entries))
	}
}

func TestLedger_NegativeBalanceTolerance(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_NEGATIVE_TOLERANCE"
	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 1000, // 10 TL bakiye var
	})

	// 20 TL biniş yapılmak isteniyor. Tolerans limiti: 50 TL (5000 kuruş)
	// Bakiye 10 TL - 20 TL = -10 TL olacak. -10 TL >= -50 TL olduğu için geçiş onaylanmalı!
	_, err := svc.ProcessTap(ctx, cardID, 2000, "VAL_NEG_1", "Negatif toleranslı biniş", 5000)
	if err != nil {
		t.Fatalf("Negatif tolerans dahilinde biniş onaylanmalıydı: %v", err)
	}

	bal, _ := svc.GetCardBalance(ctx, cardID)
	if bal != -1000 {
		t.Fatalf("Beklenen negatif bakiye -1000 kuruş, bulunan: %d", bal)
	}

	// Şimdi bakiye -10 TL. Bir 50 TL biniş daha istenirse: -10 - 50 = -60 TL (< -50 TL limit). Reddedilmeli!
	_, err = svc.ProcessTap(ctx, cardID, 5000, "VAL_NEG_2", "Limit aşımı biniş", 5000)
	if err != ErrInsufficientBalance {
		t.Fatalf("Limit aşımında yetersiz bakiye hatası bekleniyordu, alınan: %v", err)
	}
}
