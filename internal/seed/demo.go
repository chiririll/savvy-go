package seed

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"time"

	"savvy-go/internal/auth"
	appdb "savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/domain"
)

const (
	monthsOfHistory = 12
	rngSeed         = 20260605
	eurUSDRate      = 1.08
)

var merchants = map[string][]string{
	"#GROCERIES":     {"Whole Foods Market", "Trader Joe's", "Costco Wholesale", "Walmart", "Kroger", "Aldi", "Target", "Safeway"},
	"#TRANSPORT":     {"Shell", "BP", "Chevron", "Uber", "Lyft", "Metro Transit", "ParkMobile", "Jiffy Lube"},
	"#DINING":        {"Starbucks", "Chipotle Mexican Grill", "McDonald's", "Olive Garden", "Subway", "Domino's Pizza", "Panera Bread", "Five Guys"},
	"#ENTERTAINMENT": {"AMC Theatres", "Steam", "PlayStation Store", "Ticketmaster", "Netflix", "Spotify"},
	"#SHOPPING":      {"Amazon", "Best Buy", "IKEA", "Nike", "Apple Store", "Zara", "H&M", "Home Depot"},
	"#HEALTH":        {"CVS Pharmacy", "Walgreens", "City Medical Clinic", "Planet Fitness", "LA Fitness", "Quest Diagnostics"},
	"#PERSONAL_CARE": {"Supercuts", "Sephora", "Ulta Beauty", "The Barber Shop"},
	"#GIFTS":         {"Etsy", "Amazon", "Tiffany & Co.", "Local Florist"},
	"#TRAVEL":        {"Booking.com", "Airbnb", "Delta Air Lines", "Marriott", "Expedia", "Hertz"},
	"#UTILITIES":     {"ConEdison", "AT&T", "Xfinity", "National Grid", "Verizon Wireless"},
	"#HOUSING":       {"IKEA", "Home Depot", "Bed Bath & Beyond"},
	"#OTHER":         {"PayPal", "Venmo", "Cash Withdrawal", "Square"},
}

var catalogs = map[string][]string{
	"#GROCERIES":     {"Organic Bananas", "Whole Milk", "Sourdough Bread", "Free-range Eggs", "Chicken Breast", "Avocados", "Greek Yogurt", "Baby Spinach", "Ground Coffee", "Cheddar Cheese", "Pasta", "Olive Oil", "Roma Tomatoes", "Atlantic Salmon", "Brown Rice", "Butter", "Orange Juice", "Granola"},
	"#DINING":        {"Latte", "Avocado Toast", "Caesar Salad", "Burger Combo", "Iced Americano", "Chicken Bowl", "Fries", "Cheesecake", "Iced Tea", "Soup of the Day", "Fish Tacos", "Espresso"},
	"#TRANSPORT":     {"Regular Gasoline", "Premium Gasoline", "Airport Parking", "Car Wash", "Metro Day Pass", "Oil Change", "Tire Rotation", "Tolls"},
	"#SHOPPING":      {"Cotton T-Shirt", "Wireless Earbuds", "Desk Lamp", "Running Shorts", "Phone Case", "Notebook Set", "Kitchen Towels", "HDMI Cable", "Sneakers", "Backpack"},
	"#ENTERTAINMENT": {"Movie Ticket", "Popcorn Combo", "Game Download", "Arcade Tokens", "Concert Ticket", "Streaming Rental", "Base Plan"},
	"#HEALTH":        {"Prescription Refill", "Vitamin D", "Ibuprofen", "Allergy Test", "Contact Lenses", "First Aid Kit", "Protein Powder"},
	"#PERSONAL_CARE": {"Haircut", "Shampoo", "Face Moisturizer", "Toothpaste", "Sunscreen", "Nail Polish", "Beard Oil"},
	"#TRAVEL":        {"Hotel Night", "Airport Transfer", "City Museum Pass", "Travel Adapter", "Bottled Water", "Snack Box", "Souvenir"},
	"#GIFTS":         {"Greeting Card", "Gift Wrap", "Scented Candle", "Chocolate Box", "Bouquet", "Mug"},
	"#UTILITIES":     {"Electricity Usage", "Delivery Charge", "Service Fee"},
	"#HOUSING":       {"Shelf brackets", "Light bulbs", "Paint", "Towels"},
	"#RENT":          {"Base Rent"},
	"#OTHER":         {"Service Fee", "Convenience Charge", "Misc. Purchase", "Packaging"},
}

type seeder struct {
	ctx      context.Context
	db       *sql.DB
	now      time.Time
	start    time.Time
	end      time.Time
	rng      *rand.Rand
	txs      domain.Transactions
	accts    domain.Accounts
	cats     domain.Categories
	tags     domain.Tags
	budgets  domain.Budgets
	recur    domain.RecurringStore
	auto     domain.Automation
	expenses []expenseCand
}

type expenseCand struct {
	id      int64
	amount  float64
	catName string
}

func seedWorkspace(ctx context.Context, db *sql.DB, loc *time.Location) error {
	now := time.Now().In(loc)
	start := startOfMonth(now.AddDate(0, -monthsOfHistory, 0))
	s := &seeder{
		ctx:     ctx,
		db:      db,
		now:     now,
		start:   start,
		end:     now,
		rng:     rand.New(rand.NewSource(rngSeed)),
		txs:     domain.Transactions{DB: db},
		accts:   domain.Accounts{DB: db},
		cats:    domain.Categories{DB: db},
		tags:    domain.Tags{DB: db},
		budgets: domain.Budgets{DB: db},
		recur:   domain.RecurringStore{DB: db, Txs: domain.Transactions{DB: db}},
		auto:    domain.Automation{DB: db, Txs: domain.Transactions{DB: db}},
	}
	if err := s.createUsers(); err != nil {
		return err
	}
	curs := domain.Currencies{DB: db}
	usd, err := curs.ByCode(ctx, "USD")
	if err != nil || usd == nil {
		return fmt.Errorf("USD currency missing")
	}
	eur, err := curs.ByCode(ctx, "EUR")
	if err != nil {
		return err
	}
	accounts, err := s.createAccounts(usd, eur)
	if err != nil {
		return err
	}
	expenses, err := s.cats.All(ctx, "expense")
	if err != nil {
		return err
	}
	incomes, err := s.cats.All(ctx, "income")
	if err != nil {
		return err
	}
	tags, err := s.tags.All(ctx)
	if err != nil {
		return err
	}
	if err := s.createTransactions(accounts, expenses, incomes, tags); err != nil {
		return err
	}
	if err := s.seedTransactionItems(); err != nil {
		return err
	}
	if err := s.createBudgets(usd, expenses, tags); err != nil {
		return err
	}
	if err := s.createRecurring(accounts, expenses, incomes); err != nil {
		return err
	}
	return s.createAutomation(tags)
}

func (s *seeder) createUsers() error {
	users := auth.Users{DB: s.db}
	for _, u := range []struct {
		name, email, pass, role string
	}{
		{"Alex Morgan", "admin@savvy.app", "password", auth.RoleAdmin},
		{"Jordan Lee", "editor@savvy.app", "password", auth.RoleReadWrite},
		{"Demo User", "demo@demo.com", "demo", auth.RoleReadOnly},
	} {
		pass := u.pass
		if _, err := users.Create(s.ctx, u.name, u.email, &pass, u.role); err != nil {
			return fmt.Errorf("user %s: %w", u.email, err)
		}
	}
	return nil
}

func (s *seeder) createAccounts(usd *domain.Currency, eur *domain.Currency) (map[string]*domain.Account, error) {
	out := map[string]*domain.Account{}
	type spec struct {
		key, name, typ string
		cur            *domain.Currency
		bal            float64
		debtType       string
		target         *float64
		due            *string
		counter, desc  string
	}
	iowe := "i_owe"
	owed := "owed_to_me"
	mortgageDue := s.now.AddDate(25, 0, 0).Format("2006-01-02")
	loanDue := s.now.AddDate(0, 4, 0).Format("2006-01-02")
	zero := 0.0
	mortgageAmt := 285000.00
	loanAmt := 1200.00

	specs := []spec{
		{"checking", "Chase Checking", "bank", usd, 8200, "", nil, nil, "", ""},
		{"cash", "Cash Wallet", "cash", usd, 340, "", nil, nil, "", ""},
		{"savings", "Ally Savings", "bank", usd, 24500, "", nil, nil, "", ""},
		{"crypto", "Coinbase Portfolio", "crypto", usd, 6300, "", nil, nil, "", ""},
	}
	if eur != nil {
		specs = append(specs, spec{"eur", "Revolut EUR", "bank", eur, 1850, "", nil, nil, "", ""})
	}
	specs = append(specs,
		spec{"credit", "Amex Gold Card", "debt", usd, 0, iowe, &zero, nil, "American Express", "Credit card balance"},
		spec{"mortgage", "Home Mortgage", "debt", usd, 0, iowe, &mortgageAmt, &mortgageDue, "Wells Fargo Home Mortgage", "30-year fixed mortgage"},
		spec{"loan_out", "Loan to Michael", "debt", usd, 0, owed, &loanAmt, &loanDue, "Michael Chen", "Helped with moving costs"},
	)

	for _, sp := range specs {
		a := domain.Account{
			Name: sp.name, Type: sp.typ, CurrencyID: sp.cur.ID,
			InitialBalance: sp.bal, IsActive: true,
		}
		if sp.debtType != "" {
			a.DebtType = &sp.debtType
			a.TargetAmount = sp.target
			a.DueDate = sp.due
			a.Counterparty = strPtr(sp.counter)
			a.DebtDesc = strPtr(sp.desc)
		}
		created, err := s.accts.Create(s.ctx, a)
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", sp.name, err)
		}
		out[sp.key] = created
	}
	return out, nil
}

func (s *seeder) createTransactions(accounts map[string]*domain.Account, expenses, incomes []domain.Category, tags []domain.Tag) error {
	if err := s.seedIncome(accounts, incomes); err != nil {
		return err
	}
	if err := s.seedFixedExpenses(accounts, expenses); err != nil {
		return err
	}
	if err := s.seedVariableExpenses(accounts, expenses, tags); err != nil {
		return err
	}
	return s.seedTransfersAndDebt(accounts)
}

func (s *seeder) seedIncome(accounts map[string]*domain.Account, incomes []domain.Category) error {
	salary := catByName(incomes, "#SALARY")
	freelance := catByName(incomes, "#FREELANCE")
	investments := catByName(incomes, "#INVESTMENTS")
	other := catByName(incomes, "#OTHER_INCOME")
	baseSalary := 6400.0

	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		for _, payDay := range []int{5, 20} {
			payday := dateOn(cursor, minInt(payDay, daysInMonth(cursor)))
			if salary != nil && s.inRange(payday) {
				raise := 0.0
				if monthsBetween(s.start, cursor) >= 7 {
					raise = 600
				}
				if err := s.addTx("income", accounts["checking"].ID, &salary.ID,
					baseSalary+raise+float64(s.mtRand(-40, 40)),
					"Acme Corp — Payroll", payday, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}

		if freelance != nil && s.mtRand(1, 100) <= 65 {
			d := dateOn(cursor, s.mtRand(8, minInt(26, daysInMonth(cursor))))
			if s.inRange(d) {
				vendors := []string{"Upwork payout", "Fiverr withdrawal", fmt.Sprintf("Client invoice #%d", s.mtRand(1000, 9999))}
				acct := accounts["checking"]
				if accounts["eur"] != nil && s.mtRand(0, 1) == 1 {
					acct = accounts["eur"]
				}
				if err := s.addTx("income", acct.ID, &freelance.ID, float64(s.mtRand(450, 1900)),
					vendors[s.mtRand(0, 2)], d, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}

		if investments != nil && int(cursor.Month())%3 == 1 {
			d := dateOn(cursor, minInt(15, daysInMonth(cursor)))
			if s.inRange(d) {
				if err := s.addTx("income", accounts["crypto"].ID, &investments.ID,
					float64(s.mtRand(80, 420)), "Quarterly dividend payout", d, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}
	}

	if other != nil {
		d := dateOn(s.now.AddDate(0, -2, 0), 12)
		if err := s.addTx("income", accounts["checking"].ID, &other.ID,
			float64(s.mtRand(900, 1400)), "Tax refund — IRS", d, nil, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedFixedExpenses(accounts map[string]*domain.Account, expenses []domain.Category) error {
	rent := catByName(expenses, "#RENT")
	utilities := catByName(expenses, "#UTILITIES")
	entertainment := catByName(expenses, "#ENTERTAINMENT")
	subs := []struct {
		name  string
		price float64
	}{
		{"Netflix", 22.99}, {"Spotify", 11.99}, {"iCloud+", 9.99},
		{"Adobe Creative Cloud", 59.99}, {"ChatGPT Plus", 20.00}, {"GitHub", 4.00},
	}

	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		if rent != nil {
			d := startOfMonth(cursor)
			if s.inRange(d) {
				if err := s.addTx("expense", accounts["checking"].ID, &rent.ID, 2150,
					"Rent — Greystar Apartments", d, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}
		if utilities != nil {
			for _, u := range []struct {
				name   string
				lo, hi int
			}{
				{"ConEdison", 70, 180},
				{"Xfinity Internet", 79, 79},
				{"National Grid Gas", 40, 120},
			} {
				d := dateOn(cursor, minInt(s.mtRand(12, 18), daysInMonth(cursor)))
				if !s.inRange(d) {
					continue
				}
				winter := 1.0
				if m := int(cursor.Month()); m == 12 || m == 1 || m == 2 {
					winter = 1.4
				}
				amt := round2(float64(s.mtRand(u.lo, u.hi)) * winter)
				if err := s.addTx("expense", accounts["checking"].ID, &utilities.ID, amt, u.name, d, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}
		if entertainment != nil {
			for _, sub := range subs {
				d := dateOn(cursor, minInt(s.mtRand(2, 9), daysInMonth(cursor)))
				if !s.inRange(d) {
					continue
				}
				if err := s.addTx("expense", accounts["credit"].ID, &entertainment.ID, sub.price,
					sub.name+" subscription", d, nil, nil, nil, nil); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *seeder) seedVariableExpenses(accounts map[string]*domain.Account, expenses []domain.Category, tags []domain.Tag) error {
	weights := []struct {
		name string
		w    int
	}{
		{"#GROCERIES", 22}, {"#DINING", 20}, {"#TRANSPORT", 16},
		{"#SHOPPING", 12}, {"#ENTERTAINMENT", 9}, {"#HEALTH", 6}, {"#PERSONAL_CARE", 5},
		{"#TRAVEL", 3}, {"#HOUSING", 2}, {"#GIFTS", 2}, {"#OTHER", 2},
	}
	var pool []domain.Category
	for _, w := range weights {
		if c := catByName(expenses, w.name); c != nil {
			for i := 0; i < w.w; i++ {
				pool = append(pool, *c)
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	ranges := map[string][2]int{
		"#GROCERIES": {18, 145}, "#DINING": {9, 70}, "#TRANSPORT": {4, 65},
		"#SHOPPING": {15, 240}, "#ENTERTAINMENT": {12, 95}, "#HEALTH": {10, 160},
		"#PERSONAL_CARE": {15, 85}, "#TRAVEL": {120, 950}, "#HOUSING": {25, 180},
		"#GIFTS": {20, 180}, "#OTHER": {10, 120},
	}
	vacation := tagByName(tags, "Vacation")
	essential := tagByName(tags, "Essential")
	business := tagByName(tags, "Business")

	for cursor := s.start; !cursor.After(s.end); cursor = cursor.AddDate(0, 0, 1) {
		isWeekend := cursor.Weekday() == time.Saturday || cursor.Weekday() == time.Sunday
		seasonal := 1.0
		switch cursor.Month() {
		case time.November, time.December:
			seasonal = 1.5
		case time.July:
			seasonal = 1.25
		}
		lo, hi := 1, 3
		if isWeekend {
			lo, hi = 2, 5
		}
		count := int(math.Round(float64(s.mtRand(lo, hi)) * seasonal))
		for i := 0; i < count; i++ {
			cat := pool[s.rng.Intn(len(pool))]
			if cat.Name == "#TRAVEL" && s.mtRand(1, 100) > 12 {
				continue
			}
			r := ranges[cat.Name]
			if r[1] == 0 {
				r = [2]int{10, 80}
			}
			amount := round2(float64(s.mtRand(r[0]*100, r[1]*100)) / 100 * seasonal)
			merchant := s.merchantFor(cat.Name)
			acct := accounts["checking"]
			switch {
			case cat.Name == "#TRAVEL" && accounts["eur"] != nil && s.mtRand(0, 1) == 1:
				acct = accounts["eur"]
			case inList(cat.Name, "#GROCERIES", "#SHOPPING", "#TRAVEL", "#HEALTH") && s.mtRand(1, 100) <= 60:
				acct = accounts["credit"]
			case amount < 25 && s.mtRand(1, 100) <= 35:
				acct = accounts["cash"]
			}
			var tagIDs []int64
			if cat.Name == "#TRAVEL" && vacation != nil {
				tagIDs = append(tagIDs, vacation.ID)
			}
			if inList(cat.Name, "#GROCERIES", "#HEALTH") && essential != nil && s.mtRand(1, 100) <= 35 {
				tagIDs = append(tagIDs, essential.ID)
			}
			if cat.Name == "#HOUSING" && business != nil && s.mtRand(1, 100) <= 20 {
				tagIDs = append(tagIDs, business.ID)
			}
			if err := s.addTx("expense", acct.ID, &cat.ID, amount, merchant, cursor, nil, nil, nil, tagIDs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) seedTransfersAndDebt(accounts map[string]*domain.Account) error {
	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		d := dateOn(cursor, minInt(6, daysInMonth(cursor)))
		if s.inRange(d) {
			to := accounts["savings"].ID
			amt := 800.0
			if err := s.addTx("transfer", accounts["checking"].ID, nil, amt,
				"Automatic transfer to savings", d, &to, &amt, nil, nil); err != nil {
				return err
			}
		}

		dCard := dateOn(cursor, minInt(25, daysInMonth(cursor)))
		if s.inRange(dCard) {
			pay := float64(s.mtRand(600, 1400))
			to := accounts["credit"].ID
			if err := s.addTx("debt_payment", accounts["checking"].ID, nil, pay,
				"Amex statement payment", dCard, &to, &pay, nil, nil); err != nil {
				return err
			}
		}

		dMort := dateOn(cursor, minInt(2, daysInMonth(cursor)))
		if s.inRange(dMort) {
			to := accounts["mortgage"].ID
			amt := 1680.00
			if err := s.addTx("debt_payment", accounts["checking"].ID, nil, amt,
				"Monthly mortgage payment", dMort, &to, &amt, nil, nil); err != nil {
				return err
			}
		}

		if accounts["eur"] != nil && int(cursor.Month())%2 == 0 {
			dFx := dateOn(cursor, minInt(14, daysInMonth(cursor)))
			if s.inRange(dFx) {
				usdOut := float64(s.mtRand(300, 600))
				toAmt := round2(usdOut / eurUSDRate)
				rate := roundN(1/eurUSDRate, 6)
				to := accounts["eur"].ID
				if err := s.addTx("transfer", accounts["checking"].ID, nil, usdOut,
					"USD → EUR top-up", dFx, &to, &toAmt, &rate, nil); err != nil {
					return err
				}
			}
		}
	}

	d := dateOn(s.now.AddDate(0, -2, 0), 18)
	to := accounts["loan_out"].ID
	amt := 400.0
	return s.addTx("debt_collection", accounts["checking"].ID, nil, amt,
		"Michael — partial repayment", d, &to, &amt, nil, nil)
}

func (s *seeder) seedTransactionItems() error {
	if len(s.expenses) == 0 {
		return nil
	}
	target := int(math.Round(float64(s.countTx()) * 0.10))
	if target < 1 {
		target = 1
	}
	s.rng.Shuffle(len(s.expenses), func(i, j int) {
		s.expenses[i], s.expenses[j] = s.expenses[j], s.expenses[i]
	})
	if target > len(s.expenses) {
		target = len(s.expenses)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, cand := range s.expenses[:target] {
		catName := cand.catName
		if catName == "" {
			catName = "#OTHER"
		}
		items := s.randomItems(cand.amount, catName)
		for _, it := range items {
			if err := appdb.Q(s.db).InsertTransactionItem(s.ctx, sqlc.InsertTransactionItemParams{
				TransactionID: cand.id, Name: it.Name, Quantity: it.Quantity, PricePerUnit: it.PricePerUnit,
				TotalPrice: it.TotalPrice, CreatedAt: appdb.NS(now), UpdatedAt: appdb.NS(now),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) randomItems(amount float64, category string) []domain.TxItem {
	catalog := catalogs[category]
	if len(catalog) == 0 {
		catalog = catalogs["#OTHER"]
	}
	remainingCents := int(math.Round(amount * 100))
	if remainingCents < 1 {
		return nil
	}
	maxItems := minInt(6, minInt(len(catalog), remainingCents))
	count := 1
	if maxItems > 1 {
		count = s.mtRand(2, maxItems)
	}
	names := append([]string{}, catalog...)
	s.rng.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	names = names[:count]

	var items []domain.TxItem
	left := len(names)
	for _, name := range names {
		left--
		canSplit := left > 0 && remainingCents > left
		if !canSplit {
			if remainingCents < 1 {
				break
			}
			price := round2(float64(remainingCents) / 100)
			items = append(items, domain.TxItem{Name: name, Quantity: 1, PricePerUnit: price, TotalPrice: price})
			break
		}
		maxShare := remainingCents - left
		qty := s.mtRand(1, minInt(3, maxShare))
		share := maxInt(qty, (maxShare*s.mtRand(25, 55))/100)
		lineCents := (share / qty) * qty
		lineCents = maxInt(qty, minInt(lineCents, (maxShare/qty)*qty))
		remainingCents -= lineCents
		items = append(items, domain.TxItem{
			Name: name, Quantity: float64(qty),
			PricePerUnit: round2(float64(lineCents) / float64(qty) / 100),
			TotalPrice:   round2(float64(lineCents) / 100),
		})
	}
	return items
}

func (s *seeder) createBudgets(usd *domain.Currency, expenses []domain.Category, tags []domain.Tag) error {
	start := startOfMonth(s.now).Format("2006-01-02")
	type named struct {
		name   string
		amount float64
		cat    string
		notify int
	}
	for _, b := range []named{
		{"Groceries", 750, "#GROCERIES", 80},
		{"Dining Out", 450, "#DINING", 90},
		{"Fun Money", 300, "#ENTERTAINMENT", 100},
	} {
		cat := catByName(expenses, b.cat)
		if cat == nil {
			continue
		}
		notify := b.notify
		if _, err := s.budgets.Create(s.ctx, domain.BudgetInput{
			Name: b.name, Amount: b.amount, CurrencyID: &usd.ID, Period: "monthly",
			StartDate: &start, NotifyAtPercent: &notify, CategoryIDs: []int64{cat.ID},
		}); err != nil {
			return err
		}
	}
	notify := 85
	global := true
	if _, err := s.budgets.Create(s.ctx, domain.BudgetInput{
		Name: "Total Monthly Spending", Amount: 5500, CurrencyID: &usd.ID, Period: "monthly",
		StartDate: &start, IsGlobal: &global, NotifyAtPercent: &notify,
	}); err != nil {
		return err
	}
	if vac := tagByName(tags, "Vacation"); vac != nil {
		from := s.now.AddDate(0, -1, 0).Format("2006-01-02")
		to := s.now.AddDate(0, 4, 0).Format("2006-01-02")
		n := 75
		if _, err := s.budgets.Create(s.ctx, domain.BudgetInput{
			Name: "Italy Trip 2026", Amount: 3500, CurrencyID: &usd.ID, Period: "one_time",
			StartDate: &from, EndDate: &to, NotifyAtPercent: &n, TagIDs: []int64{vac.ID},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) createRecurring(accounts map[string]*domain.Account, expenses, incomes []domain.Category) error {
	if salary := catByName(incomes, "#SALARY"); salary != nil {
		day := 5
		if _, err := s.recur.Create(s.ctx, domain.RecurringInput{
			Type: "income", AccountID: accounts["checking"].ID, CategoryID: &salary.ID,
			Amount: 7000, Description: strPtr("Acme Corp — Payroll"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day,
			StartDate: nextMonthOnDay(s.now, 5).Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}
	if rent := catByName(expenses, "#RENT"); rent != nil {
		day := 1
		next := startOfMonth(s.now.AddDate(0, 1, 0)).Format("2006-01-02")
		if _, err := s.recur.Create(s.ctx, domain.RecurringInput{
			Type: "expense", AccountID: accounts["checking"].ID, CategoryID: &rent.ID,
			Amount: 2150, Description: strPtr("Rent — Greystar Apartments"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day, StartDate: next,
		}); err != nil {
			return err
		}
	}
	if subs := catByName(expenses, "#ENTERTAINMENT"); subs != nil {
		day := 4
		if _, err := s.recur.Create(s.ctx, domain.RecurringInput{
			Type: "expense", AccountID: accounts["credit"].ID, CategoryID: &subs.ID,
			Amount: 22.99, Description: strPtr("Netflix subscription"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day,
			StartDate: nextMonthOnDay(s.now, 4).Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}
	toSav := accounts["savings"].ID
	amt := 800.0
	day6 := 6
	if _, err := s.recur.Create(s.ctx, domain.RecurringInput{
		Type: "transfer", AccountID: accounts["checking"].ID, ToAccountID: &toSav,
		Amount: amt, ToAmount: &amt, Description: strPtr("Automatic transfer to savings"),
		Frequency: "monthly", Interval: 1, DayOfMonth: &day6,
		StartDate: nextMonthOnDay(s.now, 6).Format("2006-01-02"),
	}); err != nil {
		return err
	}
	toMort := accounts["mortgage"].ID
	mort := 1680.0
	day2 := 2
	if _, err := s.recur.Create(s.ctx, domain.RecurringInput{
		Type: "transfer", AccountID: accounts["checking"].ID, ToAccountID: &toMort,
		Amount: mort, ToAmount: &mort, Description: strPtr("Monthly mortgage payment"),
		Frequency: "monthly", Interval: 1, DayOfMonth: &day2,
		StartDate: nextMonthOnDay(s.now, 2).Format("2006-01-02"),
	}); err != nil {
		return err
	}
	return nil
}

func (s *seeder) createAutomation(tags []domain.Tag) error {
	trigger := "on_transaction_create"
	if essential := tagByName(tags, "Essential"); essential != nil {
		desc := "Tag any transaction above $500 as Essential"
		if _, err := s.auto.Create(s.ctx, domain.AutomationInput{
			Name: "Flag large transactions", Description: &desc, TriggerType: trigger, Priority: 1,
			Conditions: map[string]any{"match": "all", "conditions": []any{
				map[string]any{"field": "amount", "op": "gt", "value": 500},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{essential.ID}}},
		}); err != nil {
			return err
		}
	}
	if recurring := tagByName(tags, "Recurring"); recurring != nil {
		desc := `Tag transactions containing "subscription"`
		if _, err := s.auto.Create(s.ctx, domain.AutomationInput{
			Name: "Tag subscriptions", Description: &desc, TriggerType: trigger, Priority: 2,
			Conditions: map[string]any{"match": "all", "conditions": []any{
				map[string]any{"field": "description", "op": "contains", "value": "subscription"},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{recurring.ID}}},
		}); err != nil {
			return err
		}
	}
	if business := tagByName(tags, "Business"); business != nil {
		desc := "Tag GitHub / Adobe / Udemy as Business"
		if _, err := s.auto.Create(s.ctx, domain.AutomationInput{
			Name: "Tag software tools", Description: &desc, TriggerType: trigger, Priority: 3,
			Conditions: map[string]any{"match": "any", "conditions": []any{
				map[string]any{"field": "description", "op": "contains", "value": "GitHub"},
				map[string]any{"field": "description", "op": "contains", "value": "Adobe"},
				map[string]any{"field": "description", "op": "contains", "value": "Udemy"},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{business.ID}}},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) addTx(typ string, accountID int64, catID *int64, amount float64, desc string, date time.Time, toID *int64, toAmt *float64, rate *float64, tagIDs []int64) error {
	if !s.inRange(date) {
		return nil
	}
	d := date.Format("2006-01-02")
	if isFutureUTC(d) {
		return nil
	}
	status := "confirmed"
	in := domain.TxInput{
		Type: typ, AccountID: accountID, ToAccountID: toID, CategoryID: catID,
		Amount: amount, ToAmount: toAmt, ExchangeRate: rate,
		Description: &desc, Date: &d, Status: &status, TagIDs: tagIDs,
	}
	created, err := s.txs.Create(s.ctx, in)
	if err != nil {
		return fmt.Errorf("%s %s: %w", typ, desc, err)
	}
	if typ == "expense" && amount >= 2 && created != nil {
		name := ""
		if created.Category != nil {
			name = created.Category.Name
		}
		s.expenses = append(s.expenses, expenseCand{id: created.ID, amount: amount, catName: name})
	}
	return nil
}

func (s *seeder) countTx() int {
	n, _ := appdb.Q(s.db).CountTransactions(s.ctx, sqlc.CountTransactionsParams{})
	return int(n)
}

func (s *seeder) inRange(d time.Time) bool {
	day := d.Format("2006-01-02")
	return day >= s.start.Format("2006-01-02") && day <= s.end.Format("2006-01-02")
}

func (s *seeder) mtRand(lo, hi int) int {
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + s.rng.Intn(hi-lo+1)
}

func (s *seeder) merchantFor(category string) string {
	list := merchants[category]
	if len(list) == 0 {
		return "General Store"
	}
	return list[s.rng.Intn(len(list))]
}

func catByName(cats []domain.Category, name string) *domain.Category {
	for i := range cats {
		if cats[i].Name == name {
			return &cats[i]
		}
	}
	return nil
}

func tagByName(tags []domain.Tag, name string) *domain.Tag {
	for i := range tags {
		if tags[i].Name == name {
			return &tags[i]
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

func inList(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

func dateOn(t time.Time, day int) time.Time {
	return time.Date(t.Year(), t.Month(), day, 0, 0, 0, 0, t.Location())
}

func addMonthsNoOverflow(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, t.Location())
	last := daysInMonth(first)
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, t.Location())
}

func nextMonthOnDay(now time.Time, day int) time.Time {
	d := dateOn(now, minInt(day, daysInMonth(now)))
	return addMonthsNoOverflow(d, 1)
}

func monthsBetween(from, to time.Time) int {
	return (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
}

func isFutureUTC(date string) bool {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return d.After(today)
}

func round2(v float64) float64 { return roundN(v, 2) }

func roundN(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
