package main

import(
	"fmt"
	"io/ioutil"
	"./iceberg"
	"time"
)

type testVM struct{
    iceberg.IcebergVM
}

func (vm *testVM) inst_print(args []iceberg.Entity) {
    operand, _ := vm.Get_argument(args[0], iceberg.T_STR)
    fmt.Println(operand.(string))
}

func (vm *testVM) start() {
    vm.Init()
    vm.Inst_table["print"] = iceberg.InstructionDesc{ vm.inst_print, 1, }
}

func main() {
	byte_temp, err := ioutil.ReadFile("main.ib")
	if err != nil {
		fmt.Println("Load Failed")
		return
	}
	vm := testVM{iceberg.IcebergVM{}}
	vm.start()
	script := string(byte_temp)

	t0 := time.Now()
	
	bytecode := vm.Gen_bytecode(script)

	vm.Run(bytecode)

	t1 := time.Now()
	fmt.Printf("Execution time(indluding compilation): %v ms\n", int64(t1.Sub(t0) / time.Millisecond))
}
