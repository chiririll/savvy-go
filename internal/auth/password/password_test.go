package password

import "testing"

func TestDummyNeverVerifies(t *testing.T) {
	if Dummy() {
		t.Error("the dummy check verified")
	}
	h, _ := Hash("s3cret")
	empty := ""
	for name, got := range map[string]bool{
		"no hash":     CheckConstantTime(nil, "s3cret"),
		"empty hash":  CheckConstantTime(&empty, "s3cret"),
		"empty plain": CheckConstantTime(&h, ""),
		"wrong plain": CheckConstantTime(&h, "S3cret"),
	} {
		if got {
			t.Errorf("%s verified", name)
		}
	}
	if !CheckConstantTime(&h, "s3cret") {
		t.Error("the right password must still verify")
	}
}

func TestPasswordHashesCheckOut(t *testing.T) {
	h, err := Hash("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if h == "s3cret" || !Check(h, "s3cret") {
		t.Fatal("a hash must verify its own password and not be the password")
	}
	if Check(h, "S3cret") || Check(h, "") || Check("", "s3cret") || Check("not a hash", "s3cret") {
		t.Error("a wrong password, an empty one or a broken hash verified")
	}
	other, _ := Hash("s3cret")
	if other == h {
		t.Error("hashes of one password must differ (salted)")
	}
}
