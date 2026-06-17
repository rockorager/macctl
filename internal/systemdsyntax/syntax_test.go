package systemdsyntax

import (
	"reflect"
	"testing"
)

func TestSplitItems(t *testing.T) {
	got, err := SplitItems(`/bin/echo "hello world" 'single quoted' one\stwo \x41`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/bin/echo", "hello world", "single quoted", "one two", "A"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseCommandLineExpandsSpecifiers(t *testing.T) {
	got, err := ParseCommandLine(`/bin/echo %n %N %p %i %%`, Context{UnitName: "worker@blue.service"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/bin/echo", "worker@blue.service", "worker@blue", "worker", "blue", "%"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseCommandLineExpandsEnvironment(t *testing.T) {
	ctx := Context{Environment: map[string]string{"ONE": "one", "TWO": "two two", "THREE": `"three three" four`}}
	got, err := ParseCommandLine(`echo $ONE $TWO ${TWO} $THREE pre-${ONE}`, ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"echo", "one", "two", "two", "two two", "three three", "four", "pre-one"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
