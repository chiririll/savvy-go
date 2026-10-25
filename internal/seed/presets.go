package seed

// Category is a preset category. Names are translation tokens (#HOUSING) that the
// SPA translates; IsDefault marks the default of its type.
type Category struct {
	Name, Type, Icon, Color string
	IsDefault               bool
}

// DefaultCategories returns the preset categories, in display order.
func DefaultCategories() []Category { return append([]Category(nil), defaultCategories...) }

// DefaultTags returns the preset tag names (translation tokens like the categories), in display order.
func DefaultTags() []string { return append([]string(nil), defaultTags...) }

var defaultCategories = []Category{
	{"#HOUSING", "expense", "🏠", "#a78bfa", false},
	{"#UTILITIES", "expense", "⚡", "#fbbf24", false},
	{"#SUBSCRIPTIONS", "expense", "📺", "#c084fc", false},
	{"#GROCERIES", "expense", "🛒", "#4ade80", false},
	{"#EATING_OUT", "expense", "🍽️", "#fb923c", false},
	{"#TRANSPORT", "expense", "🚗", "#60a5fa", false},
	{"#HEALTH", "expense", "🏥", "#f87171", false},
	{"#FUN", "expense", "🎮", "#f472b6", false},
	{"#CLOTHES_CARE", "expense", "🛍️", "#2dd4bf", false},
	{"#GIFTS", "expense", "🎁", "#fb7185", false},
	{"#TRAVEL", "expense", "✈️", "#38bdf8", false},
	{"#EDUCATION", "expense", "🎓", "#818cf8", false},
	{"#FAMILY", "expense", "👨‍👩‍👧", "#facc15", false},
	{"#OTHER", "expense", "📌", "#94a3b8", true},
	{"#SALARY", "income", "💵", "#4ade80", false},
	{"#SIDE_INCOME", "income", "💻", "#60a5fa", false},
	{"#INVESTMENTS", "income", "📈", "#a78bfa", false},
	{"#GIFTS_RECEIVED", "income", "🎀", "#f472b6", false},
	{"#REFUNDS", "income", "↩️", "#2dd4bf", false},
	{"#OTHER_INCOME", "income", "💰", "#94a3b8", true},
}

var defaultTags = []string{"#ESSENTIAL", "#SUBSCRIPTION", "#SHARED", "#BUSINESS", "#VACATION"}
