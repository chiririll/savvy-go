package spaces

import "testing"

func TestLookValidation(t *testing.T) {
	s := func(v string) *string { return &v }
	ok := []Look{
		{},
		{Emoji: s("🏠")},
		{Emoji: s("👨‍👩‍👧‍👦")},
		{Emoji: s("🇩🇪")},
		{Emoji: s("👍🏽")},
		{Emoji: s("❤️")},
		{Emoji: s("1️⃣")},
		{Color: s("#A1b2C3")},
	}
	for _, l := range ok {
		if _, err := Normalize(l); err != nil {
			t.Errorf("%+v rejected: %v", l, err)
		}
	}
	bad := map[string]Look{
		"two emoji":   {Emoji: s("🏠🏠")},
		"text":        {Emoji: s("ab")},
		"long":        {Emoji: s("🏠 a very long string")},
		"color name":  {Color: s("red")},
		"short color": {Color: s("#fff")},
		"no hash":     {Color: s("ff8800")},
	}
	for name, l := range bad {
		_, err := Normalize(l)
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	n, _ := Normalize(Look{Color: s("#A1B2C3"), Emoji: s("  ")})
	if *n.Color != "#a1b2c3" || n.Emoji != nil {
		t.Errorf("normalize = %+v", n)
	}
}
