# QA Master Test Plan: AlpaCard Web Platform

## 1. Project Objective

The primary objective is to guarantee functional quality, reliability, and security across the **AlpaCard** web platform ([alpacard.app](https://alpacard.app)). The main goal is to ensure that the core customer loyalty lifecycle—stamp accumulation, reward redemption, and counter reset—operates without logical errors or role-based security vulnerabilities.

---

## 2. Scope

### In-Scope
- **Authentication & User Management:** User login and registration across roles (`OWNER` / Merchant, `STAFF` / Barista, and `CONSUMER` / Client).
- **Loyalty Lifecycle (Core):** Logic for incrementing stamps on virtual cards, progress visualization, unlocking reward redemption upon reaching the campaign threshold, and resetting the counter back to zero after consumption.
- **Roles & Permissions (RBAC):** Validating strict role enforcement, specifically ensuring that a user with the `CONSUMER` role cannot alter or forge their own stamp balance, and ensuring `STAFF` can only operate campaigns belonging to their merchant.

### Out-of-Scope
- Server stress and high-volume performance testing.
- Payment gateway integration testing (excluded from current release flows).

---

## 3. Test Strategy & Approach

The project testing approach is organized across three complementary execution layers to ensure comprehensive test coverage:

1. **Manual Functional Testing:**  
   Validation of critical end-to-end business flows using production-like scenarios. This includes simulating recurring customers accumulating stamps to redeem for venue rewards (such as a coffee, cheese scone, or scallion scone) until completing the card, verifying that the user interface updates and reflects state transitions in real time.
2. **Database Integrity Verification:**  
   Direct relational data verification against **PostgreSQL 16** using database management clients (such as pgAdmin, DBeaver, or TablePlus). Relational integrity checks and `JOIN` queries will verify that transaction records, stamp increments, and entity relationships persist accurately without duplicates or inconsistent states.
3. **Automated Regression Suite (Phase 2):**  
   Development of automated regression scripts utilizing **Selenium WebDriver with Java**. Test setup will implement custom `ChromeOptions` to:
   - Emulate mobile viewports and touch devices natively.
   - Mock camera permissions and web media streams (`--use-fake-ui-for-media-stream`).
   - Validate Service Worker caching mechanisms and offline PWA behavior.
   - Integrate automated test execution directly into GitHub Actions continuous integration pipelines (`ci-main.yml` / `ci-integration.yml`).

---

## 4. Environment & Tooling

| Category | Tool / Technology | Purpose |
| :--- | :--- | :--- |
| **Test Case & Bug Management** | Jira integrated with Xray | Test repository management, traceability, execution recording, and defect tracking. |
| **Database Management & Verification** | pgAdmin / DBeaver / TablePlus (PostgreSQL 16) | Direct relational integrity validation, query debugging, and data state verification. |
| **Development & Test Automation** | IntelliJ IDEA, Java, Selenium WebDriver | Automated regression script development and native mobile browser emulation. |
| **Edge Case Generation & QA Assistance** | Cursor, Claude, Antigravity | Rapid authoring of complex test matrices, edge-case analysis, and automation debugging. |
| **Continuous Integration** | GitHub Actions (`ci-main.yml` / `ci-integration.yml`) | Automated build validation and regression test pipeline execution. |

### 4.1 Database Reachability from VPN & Local Network

The PostgreSQL database instances are accessible directly over the **Tailscale VPN** mesh and the **local network** (`192.168.0.0/24`) for QA automated testing, data integrity verification, and database clients:

| Environment | Hostname (DNS) | Direct IP (LAN/VPN) | NodePort Fallback | Port | Database | Username | Password |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Development (`alpacard-dev`)** | `dev-db.alpacard.app` | `192.168.0.54` | `192.168.0.103:30542` | `5432` | `alpacard_dev` | `alpacard_dev` | `alpacard_dev_secret_2026` |
| **Production (`alpacard`)** | `prod-db.alpacard.app` | `192.168.0.55` | `192.168.0.103:30541` | `5432` | `alpacard` | `alpacard` | `alpacard_pg_prod_secret_2026` |

#### Connection String (Standard URI)
- **Development (Primary QA Target)**:  
  `postgres://alpacard_dev:alpacard_dev_secret_2026@dev-db.alpacard.app:5432/alpacard_dev?sslmode=disable`
- **NodePort Alternative**:  
  `postgres://alpacard_dev:alpacard_dev_secret_2026@192.168.0.103:30542/alpacard_dev?sslmode=disable`

### 4.2 Automated QA Testing Commands
The QA automation team can execute the test suites directly via CLI or integrated test runners:

```bash
# 1. Run Complete E2E Suite (Newman API Automation + Direct Database Integrity Verification)
pnpm test:qa:all
# or: ./scripts/qa/run-e2e-suite.sh

# 2. Run Direct Relational Database Integrity & Constraint Verification
pnpm test:qa:db
# or: ./scripts/qa/test-db.sh

# 3. Run Newman API Contract & Workflow Regression
pnpm test:qa:api
```

---

## 5. Acceptance & Suspension Criteria

### Suspension Criteria
Testing execution will be suspended if a blocker defect in the **Authentication Module** or core routing prevents general access to the platform or halts navigation into primary loyalty workflows.

### Acceptance Criteria (Exit Criteria)
The test cycle is considered complete when:
- **100% of critical test cases** (including Authentication and Stamp Issuance/Redemption flows) have been fully executed.
- **Zero open defects** of High or Critical severity remain unresolved.

---

## 6. Inventory of Functionalities

| Main Module | Sub-Module / Screen | Business Rule (Expected Result) | Role | Priority |
| :--- | :--- | :--- | :--- | :--- |
| **Backoffice** | Create Merchant | The owner registers their venue, brand colors, and menu. All data is isolated within their database tenant. | `OWNER` | Medium |
| **Backoffice** | Campaign Management | Defines the rules of the campaign (e.g., *"10 stamps = Free Cheese Scone or Coffee"*). Validates input limits. | `OWNER` | High |
| **Backoffice / Acquisition** | Print Counter QR Poster | Generates a printable counter poster pointing directly to `https://wallet.alpacard.app/claim/[campaignId]`. | `OWNER` | High |
| **Acquisition** | Counter QR Scan | Scanning the printed poster opens `https://wallet.alpacard.app/claim/[campaignId]` and provisions an anonymous wallet in under 1 second without signup forms. | `CONSUMER` | High |
| **Acquisition** | Session Persistence | Remembers the user session via cookies or LocalStorage. Reloading (F5) or reopening the browser preserves the wallet and progress. | `CONSUMER` | High |
| **Acquisition** | PWA Offline Resilience | Installed PWA loads previous card state offline from Service Worker cache rather than a generic network error screen. | `CONSUMER` | Medium |
| **Fidelity (Core)** | Stamp QR Generation | The customer's PWA generates a dynamic cryptographic QR code (signed JWT) that expires strictly in 5 minutes. | `CONSUMER` | Critical |
| **Operation** | Barista Scanner | Reads the customer's QR code. If valid, emits an audible chime, triggers vibration, and increments one stamp. | `STAFF` | Critical |
| **Operation / Fidelity** | Reward Unlock & Claim | Reaching the campaign stamp limit disables "Add Stamp", enables "Claim Reward", and resets the counter to 0 upon staff validation. | `STAFF` / `CONSUMER` | Critical |
| **Security** | Anti-Replay Protection | If staff or a malicious actor attempts to scan the same QR code twice, the API rejects it with an HTTP `409 Conflict` error. | `STAFF` / API | Critical |
| **Security** | Access Control (RBAC) | A `CONSUMER` cannot scan cards. `STAFF` can only operate campaigns belonging to their own venue. | All Roles | High |

---

## 7. Test Scenarios and Test Cases

### Test Scenario 01: Verify QR Code Security, Expiration, and Anti-Fraud Prevention

Validates that cryptographic QR tokens used for stamp allocation function securely, cannot be replayed, expire after the defined duration, and behave properly on client generation.

#### Test Cases:
- **TC-001 (Happy Path):** Validate that the QR code successfully increments a stamp when scanned for the first time (HTTP 200).
- **TC-002 (Negative Path):** Validate that the system returns HTTP `409 Conflict` if a duplicate or already used QR code is scanned.
- **TC-003 (Negative Path):** Validate that the QR code expires after exactly 5 minutes and cannot be scanned afterwards.
- **TC-004 (Edge Case):** Validate that the frontend issues a single network call to the API when generating a token, without entering infinite regeneration loops.

---

#### Detailed Specifications:

##### Test Case ID: TC-001
- **Module:** Fidelity / Operation (Frontend)
- **Scenario:** Validate that the QR code adds a stamp correctly when scanned for the first time in the interface.
- **Preconditions:**
  - Device A (Customer): Access to the AlpaCard PWA with an active anonymous wallet that has at least one open slot for a stamp.
  - Device B (Barista): Logged in with role `STAFF` or `OWNER` with access to the scanner tool using camera or web app.
- **Execution Steps:**
  1. On Device A, generate and display a fresh dynamic QR code on screen.
  2. On Device B, open the platform scanner.
  3. Aim Device B's camera at Device A's screen to read the QR code before the 5 minutes expire.
  4. Observe physical and visual feedback on Device B.
  5. Refresh or check the wallet screen on Device A.
- **Expected Result:**
  - Device B (Barista) emits a confirmation chime, vibrates, and shows a success notification indicating the stamp was added.
  - Device A (Customer) updates its interface in real time, incrementing the visual stamp counter (e.g., from `0/10` to `1/10`).

##### Test Case ID: TC-002
- **Module:** Security / Anti-Fraud Prevention
- **Scenario:** Validate that the system returns HTTP `409 Conflict` if a duplicate or already used QR code is scanned.
- **Preconditions:**
  1. Valid JWT token recently generated from customer view.
  2. Valid Bearer Token corresponding to a user with role `STAFF` or `OWNER`.
- **Execution Steps:**
  1. Configure a `POST` request in Postman/client to the `/api/v1/scan` endpoint.
  2. Provide the `STAFF` Bearer Token in authorization headers.
  3. Send the request including the customer's JWT token in the JSON body.
  4. Confirm that the system returns HTTP `200 OK`.
  5. Immediately resend the exact same `POST` request using the identical JWT token.
- **Expected Result:**
  - The second request is rejected by the backend, returning strictly an HTTP `409 Conflict` code to prevent duplicate stamp increments.

##### Test Case ID: TC-003
- **Module:** Fidelity / Security
- **Scenario:** Validate QR code expiration upon reaching the 5-minute time limit.
- **Preconditions:**
  - Device A (Customer): Access to the AlpaCard PWA with an active wallet.
  - Device B (Barista): Session logged in with role `STAFF` or `OWNER` and scanner tool open.
  - Temporarily disable automatic QR refresh (or take a screenshot as soon as generated to freeze the image).
- **Execution Steps:**
  1. On Device A, generate a new dynamic QR code and note the exact time.
  2. Deliberately wait for 5 minutes and 1 second to elapse (the visual countdown timer should reach `0:00`).
  3. On Device B, open the scanner and aim at the expired QR code (or the screenshot).
  4. Observe the response on Device B and verify stamp count on Device A.
- **Expected Result:**
  - Device B's scanner rejects the code and displays an explicit error message (e.g., *"Expired token"* or *"Invalid code"*).
  - No confirmation chime or vibration is emitted.
  - The customer's stamp counter on Device A remains intact without adding any point.

##### Test Case ID: TC-004
- **Module:** Fidelity / QR Generation
- **Scenario:** Verify single network call on QR code generation and timer behavior.
- **Execution Steps:**
  1. Open browser DevTools (Network tab).
  2. Enter the customer wallet view.
  3. Validate the count of network calls made to `/qr`.
  4. Wait for the 5-minute interval to reach `0:00`.
- **Expected Result:**
  - Exactly 1 network call to `/qr` is logged.
  - The countdown timer reaches `0:00` at exactly 5 minutes, and only then is a second automatic request generated to renew the QR code.

---

### Test Scenario 02: Verify Reward Redemption Flow and Card Reset upon Reaching Limit

Validates that once customer stamps reach campaign capacity, redemption mode is unlocked, invalid additional stamps are blocked, and consumption properly resets the counter.

#### Test Cases:
- **TC-005 (Happy Path):** Validate that the interface unlocks "Reward Ready / Claim" state upon reaching the campaign stamp limit.
- **TC-006 (Happy Path):** Validate successful reward redemption by the barista and that the customer wallet counter resets to zero.
- **TC-007 (Edge Case):** Validate that the system prevents adding an additional stamp if the card is already full and pending redemption.

---

#### Detailed Specifications:

##### Test Case ID: TC-005
- **Module:** Fidelity / Reward Claim
- **Scenario:** Validate that the interface unlocks reward claim status upon reaching the stamp limit.
- **Preconditions:**
  1. Device A (Customer): Active wallet with $N-1$ stamps (e.g., 9 out of 10 stamps, missing only one).
  2. Device B (Barista): Session logged in with role `STAFF`.
- **Execution Steps:**
  1. On Device A, generate the QR code.
  2. On Device B, scan the QR code to grant the final stamp.
  3. Observe screen update on Device A.
- **Expected Result:**
  - The counter reaches the campaign maximum (e.g., `10/10`).
  - The customer's interface visually transitions and disables "Add Stamp", showing instead the option or dynamic QR code for "Claim Reward".

##### Test Case ID: TC-006
- **Module:** Fidelity / Reward Claim
- **Scenario:** Validate reward consumption and card reset.
- **Preconditions:**
  1. Device A (Customer): Full card in "Ready for claim" state (`10/10`).
  2. Device B (Barista): Session logged in with role `STAFF`.
- **Execution Steps:**
  1. On Device A, open the "Claim Reward" option and present the QR or validation view to the barista.
  2. On Device B, confirm or scan the reward redemption.
  3. Verify the resulting state on Device A.
- **Expected Result:**
  - Device B confirms that the reward was handed out.
  - Device A (via real-time SSE update) triggers a celebratory animation, screen brightness resets to normal, and the card empties back to `0`, ready for a new campaign cycle.

##### Test Case ID: TC-007
- **Module:** Fidelity / Operation (Limits)
- **Scenario:** Validate that the system blocks adding an additional stamp if the card is already full.
- **Preconditions:**
  1. Device A (Customer): Full card in "Ready for claim" state (`10/10`).
- **Execution Steps:**
  1. Attempt to force-generate a regular QR code to add a stamp on Device A (if interface permits).
  2. If generated, attempt to scan it using Device B (Barista).
  3. Observe the system response.
- **Expected Result:**
  - The system blocks the action. If the client generates the QR anyway, the barista scanner displays an error (e.g., *"Card is already full, process reward claim first"*), and the counter does not advance to `11/10`.

---

### Test Scenario 03: Verify Access, Creation, and Persistence of Anonymous Wallets for New Customers

Validates frictionless zero-form customer onboarding, session retention, error handling, and offline availability.

#### Test Cases:
- **TC-008 (Happy Path):** Validate instant anonymous wallet creation upon accessing a valid campaign URL.
- **TC-009 (Edge Case):** Validate anonymous session persistence when refreshing the page or restarting the browser (PWA).
- **TC-010 (Negative Path):** Validate error handling when attempting to access an invalid or non-existent campaign URL.
- **TC-015 (Offline PWA):** Validate card rendering from Service Worker cache without an internet connection.

---

#### Detailed Specifications:

##### Test Case ID: TC-008
- **Module:** Acquisition / Customer Wallet
- **Scenario:** Validate successful anonymous wallet creation upon accessing a valid campaign.
- **Preconditions:**
  1. Valid campaign claim link (`https://wallet.alpacard.app/claim/[campaignId]`) previously created by an `OWNER`.
  2. Use a browser in incognito mode (to simulate a brand new device with no AlpaCard history).
- **Execution Steps:**
  1. Paste the campaign URL into the browser and press Enter (simulating scanning the printed counter QR).
  2. Observe load time and initial screen presented to the user.
- **Expected Result:**
  - The system displays no registration forms and does not ask for an email.
  - Instantly redirects to the campaign loyalty card view with the counter at `0 stamps` in under 1 second.

##### Test Case ID: TC-009
- **Module:** Acquisition / PWA Persistence
- **Scenario:** Validate anonymous wallet persistence to prevent loss of customer stamps.
- **Preconditions:**
  1. Successfully completed `TC-008` and on the newly created card screen.
- **Execution Steps:**
  1. Refresh the browser page (`F5`).
  2. Close the browser tab entirely, open a new tab, and navigate back to `alpacard.app`.
- **Expected Result:**
  - The system remembers the user (via cookies or LocalStorage).
  - Exactly the same wallet loads, preserving the same ID and progress, without creating a duplicate account or prompting for re-authentication.

##### Test Case ID: TC-010
- **Module:** Acquisition / Error Handling
- **Scenario:** Validate system response when providing an invalid campaign ID.
- **Preconditions:**
  1. Browser with no active session.
- **Execution Steps:**
  1. Manually enter a malformed claim URL (e.g., `https://wallet.alpacard.app/claim/12345-fake-campaign`).
  2. Observe the interface response.
- **Expected Result:**
  - The system does not create a wallet.
  - Displays a friendly error screen (e.g., *"Campaign not found"* or *"Invalid QR Code"* - HTTP 404), blocking access to the card view.

##### Test Case ID: TC-015
- **Module:** Acquisition / Connectivity (Offline PWA)
- **Scenario:** Validate card rendering from Service Worker cache without internet connection.
- **Preconditions:**
  1. Device A (Customer): Active wallet previously loaded with network connectivity (e.g., with 3 stamps).
  2. Device placed in Airplane Mode (Wi-Fi and mobile data completely disabled).
- **Execution Steps:**
  1. Close the browser or installed PWA on Device A.
  2. Reopen the application or navigate to the card URL without internet access.
  3. Observe interface loading and displayed data.
- **Expected Result:**
  - The application loads using Service Worker cache.
  - The customer can view their previous card state (3 stamps) and graphic interface intact, rather than a generic browser "No internet connection" error.

---

### Test Scenario 04: Verify Merchant Account Creation and Campaign Rule Configuration from Backoffice

Validates merchant onboarding, loyalty rule definitions, poster generation, and strict boundary input validation.

#### Test Cases:
- **TC-011 (Happy Path):** Validate successful registration of a commercial venue defining name, colors, and menu.
- **TC-012 (Happy Path):** Validate creation of a new loyalty campaign configuring required stamp count and reward title.
- **TC-013 (Happy Path):** Validate correct generation and preview of the printable counter QR poster.
- **TC-014 (Edge Case / Negative):** Validate that the campaign creation form rejects invalid stamp counts (e.g., 0, negative numbers, or text).

---

#### Detailed Specifications:

##### Test Case ID: TC-011
- **Module:** Backoffice / Venue Configuration
- **Scenario:** Validate initial registration of a merchant account.
- **Preconditions:**
  1. Access to the AlpaCard Console with an `OWNER` role user.
- **Execution Steps:**
  1. Navigate to the venue configuration section.
  2. Complete required fields: Business Name, Brand Colors, and Menu Information.
  3. Save changes.
  4. Check venue preview or refresh page.
- **Expected Result:**
  - Data is saved to the PostgreSQL database.
  - The interface updates the admin dashboard displaying the business name and selected brand styling.

##### Test Case ID: TC-012
- **Module:** Backoffice / Campaign Management
- **Scenario:** Validate creation of a campaign with valid rules.
- **Preconditions:**
  1. Merchant configured previously (`TC-011`).
  2. User authenticated as `OWNER`.
- **Execution Steps:**
  1. Navigate to **Campaigns -> New Campaign**.
  2. Enter a valid required stamp count (e.g., `10`).
  3. Enter customer reward name (e.g., *"Free Cheese Scone"*).
  4. Save and activate campaign.
- **Expected Result:**
  - Campaign is created with status "Active" and assigned a unique `campaignId`.
  - Appears listed in the merchant's campaign dashboard.

##### Test Case ID: TC-013
- **Module:** Backoffice / Acquisition
- **Scenario:** Validate static counter QR poster generation.
- **Preconditions:**
  1. Have at least one active campaign (`TC-012`).
- **Execution Steps:**
  1. From the active campaign card, click "Print QR Poster".
  2. Inspect the generated file or popup view.
- **Expected Result:**
  - The system generates a printable view containing a static QR code.
  - Scanning this QR code with any camera decodes to the exact URL: `https://wallet.alpacard.app/claim/[campaignId]`.

##### Test Case ID: TC-014
- **Module:** Backoffice / Input Validations
- **Scenario:** Validate error handling when inputting invalid stamp counts.
- **Preconditions:**
  1. User authenticated as `OWNER` on the New Campaign screen.
- **Execution Steps:**
  1. Attempt to create a campaign entering `0` as required stamps.
  2. Click Save.
  3. Repeat by entering negative numbers (e.g., `-5`) or text (e.g., `"ten"`).
- **Expected Result:**
  - The form applies both frontend and backend validations.
  - Blocks campaign creation, highlighting the field in red and showing an explicit error message (e.g., *"Stamp count must be greater than 0"*).
