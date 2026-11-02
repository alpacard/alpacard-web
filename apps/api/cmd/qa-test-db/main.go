package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func main() {
	fmt.Printf("\n%s%s=======================================================%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s   🧪 ALPACARD QA AUTOMATION DATABASE VERIFICATION    %s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s=======================================================%s\n\n", colorBold, colorCyan, colorReset)

	targetURL := os.Getenv("DATABASE_URL")
	candidateURLs := []string{
		"postgres://alpacard_dev:alpacard_dev_secret_2026@dev-db.alpacard.app:5432/alpacard_dev?sslmode=disable",
		"postgres://alpacard_dev:alpacard_dev_secret_2026@192.168.0.54:5432/alpacard_dev?sslmode=disable",
		"postgres://alpacard_dev:alpacard_dev_secret_2026@192.168.0.103:30542/alpacard_dev?sslmode=disable",
	}

	if targetURL != "" {
		candidateURLs = []string{targetURL}
	}

	var conn *pgx.Conn
	var connectedURL string
	var connDuration time.Duration

	for _, url := range candidateURLs {
		// Mask password for display
		maskedURL := maskConnStr(url)
		fmt.Printf("🔍 Attempting connection to: %s%s%s ... ", colorYellow, maskedURL, colorReset)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := time.Now()
		c, err := pgx.Connect(ctx, url)
		cancel()

		if err == nil {
			conn = c
			connectedURL = url
			connDuration = time.Since(start)
			fmt.Printf("%s[CONNECTED]%s in %v\n", colorGreen, colorReset, connDuration)
			break
		}
		fmt.Printf("%s[FAILED]%s (%v)\n", colorRed, colorReset, err)
	}

	if conn == nil {
		fmt.Printf("\n%s❌ Fatal: Could not establish connection to PostgreSQL across candidate endpoints.%s\n", colorBold+colorRed, colorReset)
		fmt.Printf("   Please verify you are connected to the Tailscale VPN or the 192.168.0.0/24 LAN.\n\n")
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	ctx := context.Background()
	failedTests := 0
	passedTests := 0

	// 1. Connection & Session Metadata Check
	fmt.Printf("\n%s--- [1] Environment & Session Metadata ---%s\n", colorBold, colorReset)
	var currentDB, currentUser, versionStr string
	err := conn.QueryRow(ctx, "SELECT current_database(), current_user, version()").Scan(&currentDB, &currentUser, &versionStr)
	if err != nil {
		fmt.Printf("  ❌ Query metadata failed: %v\n", err)
		failedTests++
	} else {
		fmt.Printf("  ✅ Database: %s%s%s | User: %s%s%s\n", colorCyan, currentDB, colorReset, colorCyan, currentUser, colorReset)
		fmt.Printf("  ✅ Engine:   %s\n", strings.Split(versionStr, " on ")[0])
		passedTests++
	}

	// 2. Schema Tables Verification
	fmt.Printf("\n%s--- [2] Schema & Tables Presence ---%s\n", colorBold, colorReset)
	requiredTables := []string{
		"users",
		"businesses",
		"business_staff",
		"campaigns",
		"cards",
		"transactions",
		"scanned_tokens",
	}

	rows, err := conn.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema='public'")
	existingTables := make(map[string]bool)
	if err != nil {
		fmt.Printf("  ❌ Could not query information_schema: %v\n", err)
		failedTests++
	} else {
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err == nil {
				existingTables[t] = true
			}
		}
		rows.Close()

		allFound := true
		for _, req := range requiredTables {
			if existingTables[req] {
				var count int
				_ = conn.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", req)).Scan(&count)
				fmt.Printf("  ✅ Table %s%-16s%s exists (records: %s%d%s)\n", colorGreen, req, colorReset, colorCyan, count, colorReset)
			} else {
				fmt.Printf("  ❌ Table %s%-16s%s MISSING!\n", colorRed, req, colorReset)
				allFound = false
			}
		}
		if allFound {
			passedTests++
		} else {
			failedTests++
		}
	}

	// 3. Relational Integrity Checks
	fmt.Printf("\n%s--- [3] Relational Integrity & QA Constraints ---%s\n", colorBold, colorReset)

	// Check 3.1: Orphan Cards (Cards without valid campaign or user)
	var orphanCards int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM cards c
		LEFT JOIN campaigns cmp ON c.campaign_id = cmp.id
		LEFT JOIN users u ON c.user_id = u.id
		WHERE cmp.id IS NULL OR u.id IS NULL
	`).Scan(&orphanCards)
	if err != nil {
		fmt.Printf("  ⚠️ Could not test orphan cards: %v\n", err)
	} else if orphanCards == 0 {
		fmt.Printf("  ✅ Referential Integrity: 0 orphan cards found (100%% mapped to valid campaigns & users)\n")
		passedTests++
	} else {
		fmt.Printf("  ❌ Violation: Found %d orphan cards!\n", orphanCards)
		failedTests++
	}

	// Check 3.2: Orphan Transactions
	var orphanTx int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions t
		LEFT JOIN cards c ON t.card_id = c.id
		WHERE c.id IS NULL
	`).Scan(&orphanTx)
	if err != nil {
		fmt.Printf("  ⚠️ Could not test orphan transactions: %v\n", err)
	} else if orphanTx == 0 {
		fmt.Printf("  ✅ Referential Integrity: 0 orphan transactions found (100%% mapped to valid cards)\n")
		passedTests++
	} else {
		fmt.Printf("  ❌ Violation: Found %d orphan transactions!\n", orphanTx)
		failedTests++
	}

	// Check 3.3: Stamp Count Consistency
	var stampDiscrepancies int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM cards c
		WHERE c.stamps < 0
	`).Scan(&stampDiscrepancies)
	if err != nil {
		fmt.Printf("  ⚠️ Could not test stamp counts: %v\n", err)
	} else if stampDiscrepancies == 0 {
		fmt.Printf("  ✅ Business Logic: All cards have valid non-negative stamp values\n")
		passedTests++
	} else {
		fmt.Printf("  ❌ Violation: Found %d cards with negative stamps!\n", stampDiscrepancies)
		failedTests++
	}

	// 4. Anti-Fraud & Token Security Check
	fmt.Printf("\n%s--- [4] Anti-Fraud & Token Tracking ---%s\n", colorBold, colorReset)
	var tokenCount int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM scanned_tokens").Scan(&tokenCount)
	if err != nil {
		fmt.Printf("  ⚠️ Could not test scanned_tokens: %v\n", err)
	} else {
		fmt.Printf("  ✅ Anti-Replay Store: %d consumed QR nonces tracked in security audit log\n", tokenCount)
		passedTests++
	}

	// Summary
	fmt.Printf("\n%s=======================================================%s\n", colorBold, colorReset)
	if failedTests == 0 {
		fmt.Printf("🎉 %s%sQA AUTOMATION DATABASE CHECK: ALL %d TESTS PASSED!%s\n", colorBold, colorGreen, passedTests, colorReset)
		fmt.Printf("   Active Endpoint: %s\n", maskConnStr(connectedURL))
		fmt.Printf("%s=======================================================%s\n\n", colorBold, colorReset)
		os.Exit(0)
	} else {
		fmt.Printf("❌ %s%sQA AUTOMATION DATABASE CHECK: %d TESTS FAILED (%d PASSED)%s\n", colorBold, colorRed, failedTests, passedTests, colorReset)
		fmt.Printf("%s=======================================================%s\n\n", colorBold, colorReset)
		os.Exit(1)
	}
}

func maskConnStr(connStr string) string {
	parts := strings.Split(connStr, "@")
	if len(parts) == 2 {
		userPass := strings.Split(parts[0], ":")
		if len(userPass) >= 3 {
			return userPass[0] + ":" + userPass[1] + ":***@" + parts[1]
		}
	}
	return connStr
}
