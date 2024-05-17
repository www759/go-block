package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStack(t *testing.T) {
	s := NewStack(128)
	s.Push(1)
	s.Push(2)

	assert.Equal(t, s.Pop(), 1)
	assert.Equal(t, s.Pop(), 2)
	assert.Nil(t, s.Pop())
}

func TestVMSub(t *testing.T) {
	data := []byte{0x02, 0x0a, 0x1, 0x0a, 0x0e}
	contractState := NewState()
	vm := NewVM(data, contractState)
	assert.Nil(t, vm.Run())
	assert.Equal(t, -1, vm.stack.Pop())
}

func TestVMStore(t *testing.T) {
	// F O O => pack[F, O, O] => 2 => store
	// key: "FOO", value: 2

	data := []byte{0x02, 0x0a, 0x03, 0x0a, 0x0b, 0x4f, 0x0c, 0x4f, 0x0c, 0x46, 0x0c, 0x03, 0x0a, 0x0d, 0x0f}
	pushFoo := []byte{0x4f, 0x0c, 0x4f, 0x0c, 0x46, 0x0c, 0x03, 0x0a, 0x0d, 0xae}
	data = append(data, pushFoo...)

	contractState := NewState()
	vm := NewVM(data, contractState)
	assert.Nil(t, vm.Run())
	//valueBytes, err := contractState.Get([]byte("FOO"))
	//assert.Nil(t, err)
	//assert.Equal(t, DeserializeInt64(valueBytes), int64(5))
	fmt.Printf("%+v", vm.stack.data)
	fmt.Printf("%+v", contractState)
}

func TestVMMul(t *testing.T) {
	data := []byte{0x02, 0x0a, 0x02, 0x0a, 0xea}

	contractState := NewState()
	vm := NewVM(data, contractState)
	assert.Nil(t, vm.Run())

	assert.Equal(t, vm.stack.Pop(), 4)
}

func TestVMDiv(t *testing.T) {
	data := []byte{0x02, 0x0a, 0x04, 0x0a, 0xfd}

	contractState := NewState()
	vm := NewVM(data, contractState)
	assert.Nil(t, vm.Run())

	assert.Equal(t, vm.stack.Pop(), 2)
}
