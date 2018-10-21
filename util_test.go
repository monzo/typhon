package typhon

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	Client = Service(BareClient).Filter(ErrorFilter)
	os.Exit(m.Run())
}
