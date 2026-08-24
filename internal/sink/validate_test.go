package sink

import "testing"

func TestPathAndHas(t *testing.T) {
	if Path("/", "blocks") != "/blocks" {
		t.Fatal(Path("/", "blocks"))
	}
	if At("/blocks", 2) != "/blocks/2" {
		t.Fatal(At("/blocks", 2))
	}
	var p Problems
	p.Add("/text", "must be 40000 characters or fewer")
	v := p.Result()
	if v.Valid || !v.Has("/text", "40000") {
		t.Fatalf("%+v", v)
	}
}
