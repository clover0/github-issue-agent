package assert

import "testing"

func Equal[V comparable](t *testing.T, got, expected V) {
	t.Helper()

	if expected != got {
		t.Errorf(`assert.Equal(
got: %v
expected: %v
)`, got, expected)
	}
}

func Nil(t *testing.T, value any) {
	t.Helper()

	if value != nil {
		t.Errorf(`execpted nil, got %v`, value)
	}
}

func HasError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Errorf(`expected error, got nil`)
	}
}
