package seed

// Category is a preset category. Names are translation tokens (#RENT) that the
// SPA translates; IsDefault marks the default of its type.
type Category struct {
	Name, Type, Icon, Color string
	IsDefault               bool
}

// DefaultCategories returns the preset categories, in display order.
func DefaultCategories() []Category { return append([]Category(nil), defaultCategories...) }

// DefaultTags returns the preset tag names, in display order.
func DefaultTags() []string { return append([]string(nil), defaultTags...) }

var defaultCategories = []Category{
	{"#RENT", "expense", "🏢", "#94a3b8", false},
	{"#HOUSING", "expense", "🏠", "#a78bfa", false},
	{"#UTILITIES", "expense", "⚡", "#fbbf24", false},
	{"#GROCERIES", "expense", "🛒", "#4ade80", false},
	{"#TRANSPORT", "expense", "🚗", "#60a5fa", false},
	{"#HEALTH", "expense", "🏥", "#f87171", false},
	{"#DINING", "expense", "🍽️", "#fb923c", false},
	{"#ENTERTAINMENT", "expense", "🎮", "#f472b6", false},
	{"#SHOPPING", "expense", "🛍️", "#2dd4bf", false},
	{"#PERSONAL_CARE", "expense", "✨", "#e879f9", false},
	{"#GIFTS", "expense", "🎁", "#fb7185", false},
	{"#TRAVEL", "expense", "✈️", "#38bdf8", false},
	{"#OTHER", "expense", "📌", "#94a3b8", true},
	{"#SALARY", "income", "💵", "#4ade80", false},
	{"#FREELANCE", "income", "💻", "#60a5fa", false},
	{"#INVESTMENTS", "income", "📈", "#a78bfa", false},
	{"#GIFTS_RECEIVED", "income", "🎀", "#f472b6", false},
	{"#REFUNDS", "income", "↩️", "#2dd4bf", false},
	{"#OTHER_INCOME", "income", "💰", "#94a3b8", true},
}

var defaultTags = []string{
	"Essential", "Optional", "Recurring", "One-time", "Business",
	"Personal", "Family", "Vacation", "Emergency", "Planned",
}
