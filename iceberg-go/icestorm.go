// Icestorm - Iceberg implemented with Go

package iceberg

import(
	"fmt"
	"os"
	"io"
	"strings"
    "strconv"
	"bytes"
	"encoding/binary"
	"reflect"
	"math"
)

// Iceberg Types
const(
	T_UNDET int64 = 0
	T_INT   int64 = 1
	T_FLOAT int64 = 2
	T_BOOL  int64 = 4
	T_STR   int64 = 8
	T_LABEL int64 = 16

	T_ANY   int64 = 31
)

type Entity struct {
	Data []byte
	E_type int64
}

type instruction struct {
	Inst string
	Args []Entity
}

type InstructionDesc struct {
	Function func([]Entity)
	N_args int64
}

type Bytecode struct {
	inst_list []instruction
	label_table map[string]int64
}

type IcebergVM struct {
	exec_pos int64
	Inst_table map[string]InstructionDesc
	label_table map[string]int64
	var_table map[string]Entity
}

func (vm *IcebergVM) Read_str(str string) io.Reader {
	return strings.NewReader(str)
}

func (vm *IcebergVM) compile_error(message string) {
	fmt.Printf("In line %d,\n%s\n", vm.exec_pos + 1, message)
	os.Exit(1)
}
func (vm *IcebergVM) Runtime_error(message string) {
	fmt.Printf("\nIceberg runtime ERROR!\nIn instruction number %d,\n%s\n", vm.exec_pos, message)
	os.Exit(1)
}
func (vm *IcebergVM) Runtime_warning(message string) {
	fmt.Printf("\nWARNING:\nIn instruction number %d,\n%s\n", vm.exec_pos, message)
}

func (vm *IcebergVM) chk_nargs(args []Entity, expected_nargs int64) {
	n_elements := int64(len(args))
	if n_elements > expected_nargs {
		vm.compile_error(fmt.Sprintf("Syntax ERROR: Too many arguments(%d expected but %d given)", expected_nargs, n_elements))
	} else if n_elements < expected_nargs {
		vm.compile_error(fmt.Sprintf("Syntax ERROR: Too few arguments(%d expected but %d given)", expected_nargs, n_elements))
	} else {
		// number of arguments is correct, doing nothing...
	}
}

func (vm *IcebergVM) conv_arg(bs_arg []byte) Entity {
	arg_str := string(bs_arg)
	buf := new(bytes.Buffer)

	// Label?
	if strings.IndexRune(arg_str, '@') == 0 {
		err := binary.Write(buf, binary.LittleEndian, []byte(arg_str))
		if err != nil {
			vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_LABEL,
		}
	}
	// String?
	if strings.IndexRune(arg_str, '"') == 0 || strings.IndexRune(arg_str, '\'') == 0 {
		b_arg_str := []byte(arg_str)
		err := binary.Write(buf, binary.LittleEndian, b_arg_str[1:len(b_arg_str)-1])
		if err != nil {
			vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_STR,
		}
	}
	
	// Boolean?
	if arg_str == "true" || arg_str == "false" {
		temp_arg := arg_str == "true"
		err := binary.Write(buf, binary.LittleEndian, temp_arg)
		if err != nil {
			vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_BOOL,
		}
	}

	// Int?
	int_arg, err_a := strconv.ParseInt(arg_str, 10, 64)
	if err_a == nil {
		err := binary.Write(buf, binary.LittleEndian, int_arg)
		if err != nil {
			vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_INT,
		}
	}
	// Float?
	float_arg, err_b := strconv.ParseFloat(arg_str, 64)
	if err_b == nil {
		err := binary.Write(buf, binary.LittleEndian, float_arg)
		if err != nil {
			vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_FLOAT,
		}
	}

	err := binary.Write(buf, binary.LittleEndian, []byte(arg_str))
	if err != nil {
		vm.compile_error(fmt.Sprintf("System ERROR: conv_arg() failed. err: %s", err.Error()))
	}
	return Entity{
		buf.Bytes(),
		T_UNDET,
	}
}

func (vm *IcebergVM) parse_args(line string) []Entity {
	args := make([]Entity, 0)
	buf := make([]byte, len(line))
	buf_idx := 0
	d_quote := false
	s_quote := false
	after_parentheses := false

	for _, c := range line {
		if d_quote {
			if c == '"' {
				buf[buf_idx] = byte('"')
				buf_idx++
				args = append(args, vm.conv_arg(buf[:buf_idx]))
				buf_idx = 0
				d_quote = false
				after_parentheses = true
			} else {
				buf[buf_idx] = byte(c)
				buf_idx++
			}
		} else if s_quote {
			if c == '\'' {
				buf[buf_idx] = byte('\'')
				buf_idx++
				args = append(args, vm.conv_arg(buf[:buf_idx]))
				buf_idx = 0
				s_quote = false
				after_parentheses = true
			} else {
				buf[buf_idx] = byte(c)
				buf_idx++
			}
		} else {
			if c == ',' {
				if !after_parentheses {
					args = append(args, vm.conv_arg(buf[:buf_idx]))
					buf_idx = 0
				} else {
					after_parentheses = false
				}
			} else if c == '"' {
				if buf_idx != 0 {
					vm.compile_error(`Syntax ERROR: Expected , before "`)
				}
				buf[buf_idx] = byte('"')
				buf_idx++
				d_quote = true
			} else if c == '\'' {
				if buf_idx != 0 {
					vm.compile_error(`Syntax ERROR: Expected , before '`)
				}
				buf[buf_idx] = byte('\'')
				buf_idx++
				s_quote = true
			} else if c != ' ' {
				if !after_parentheses {
					buf[buf_idx] = byte(c)
					buf_idx++
				}
			}
		}
	}
	if buf_idx != 0 {
		args = append(args, vm.conv_arg(buf[:buf_idx]))
	}

	if d_quote {
		vm.compile_error(`Syntax ERROR: Missing "`)
	}
	if s_quote {
		vm.compile_error(`Syntax ERROR: Missing '`)
	}

	return args
}

func (vm *IcebergVM) parse_oneline(line string, program []instruction) []instruction {
	new_program := make([]instruction, len(program))
	copy(new_program, program)
	if strings.IndexRune(line, ' ') == -1 {
		instr := line
		_, ok := vm.Inst_table[instr]
		if ok {
			vm.chk_nargs([]Entity{}, vm.Inst_table[instr].N_args)
			new_program = append(new_program, instruction{
				instr,
				[]Entity{},
			})
		} else if strings.IndexRune(line, '@') == 0 {
			new_program = append(new_program, instruction{
				instr,
				[]Entity{},
			})
		} else if line != "" {
			vm.compile_error(fmt.Sprintf("Syntax ERROR: Unknown instruction %s", instr))
		}
	} else {
		sep_line := strings.SplitN(line, " ", 2)
		instr := sep_line[0]
		_, ok := vm.Inst_table[instr]
		if ok {
			args := vm.parse_args(sep_line[1])
			vm.chk_nargs(args, vm.Inst_table[instr].N_args)
			new_program = append(new_program, instruction{
				instr,
				args,
			})
		} else if strings.IndexRune(instr, '@') == 0 {
			vm.compile_error("Syntax ERROR: Expected newline after label definition")
		} else {
			vm.compile_error(fmt.Sprintf("Syntax ERROR: Unknown instruction %s", instr))
		}
	}
	return new_program
}

func (vm *IcebergVM) set_labels(program []instruction) ([]instruction, map[string]int64) {
	new_program := make([]instruction, len(program))
	copy(new_program, program)
	label_table := make(map[string]int64)
	for i, instr := range program {
		if strings.IndexRune(instr.Inst, '@') == 0 {
			label_table[instr.Inst] = int64(i)
			new_program[i].Inst = "nop"
		}
	}
	return new_program, label_table
}

func (vm *IcebergVM) parse_script(script string) ([]instruction, map[string]int64) {
	program := make([]instruction, 0)

	lines := strings.Split(script, "\n")
	for i, line := range lines {
		vm.exec_pos = int64(i)
		line = strings.TrimLeftFunc(line, func(c rune) bool { return c == '\n' || c == '\t' || c == ' '})
		program = vm.parse_oneline(line, program)
	}
	return vm.set_labels(program)
}

func (vm *IcebergVM) Gen_bytecode(script string) Bytecode {
	program, label_table := vm.parse_script(script)
	return Bytecode{
		program,
		label_table,
	}
}

func (vm *IcebergVM) Get_argument(arg Entity, type_mask int64) (interface{}, int64) {
	buf := bytes.NewReader(arg.Data)
	if arg.E_type == T_UNDET {
		b_symbol := make([]byte, len(arg.Data))
		err := binary.Read(buf, binary.LittleEndian, &b_symbol)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		sym_value, exist := vm.var_table[string(b_symbol)]
		if !exist {
			vm.Runtime_error(fmt.Sprintf("Argument ERROR: Unbound symbol %s", string(b_symbol)))
		}
		return vm.Get_argument(sym_value, type_mask)
	}
	
	if arg.E_type & type_mask == 0 {
		vm.Runtime_error("Type ERROR: Type mismatch")
	}
	switch arg.E_type {	
	case T_INT:
		var ret_int int64
		err := binary.Read(buf, binary.LittleEndian, &ret_int)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		return ret_int, T_INT
	case T_FLOAT:
		var ret_float float64
		err := binary.Read(buf, binary.LittleEndian, &ret_float)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		return ret_float, T_FLOAT
	case T_BOOL:
		var ret_bool bool
		err := binary.Read(buf, binary.LittleEndian, &ret_bool)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		return ret_bool, T_BOOL
	case T_STR:
		ret_b_str := make([]byte, len(arg.Data))
		err := binary.Read(buf, binary.LittleEndian, &ret_b_str)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		return string(ret_b_str), T_STR
	case T_LABEL:
		ret_b_label := make([]byte, len(arg.Data))
		err := binary.Read(buf, binary.LittleEndian, &ret_b_label)
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
		}
		return string(ret_b_label), T_LABEL
	default:
		vm.Runtime_error(fmt.Sprintf("System ERROR: Unknown typeid %d. Maybe incompatible bytecode?", arg.E_type))
	}
	// It should not happen
	return nil, T_UNDET
}
func (vm *IcebergVM) Get_baresymbol(value Entity) string{
	if value.E_type != T_UNDET {
		vm.Runtime_error("Type ERROR: Type mismatch")
	}
	buf := bytes.NewReader(value.Data)
	ret_b_sym := make([]byte, len(value.Data))
	err := binary.Read(buf, binary.LittleEndian, &ret_b_sym)
	if err != nil {
		vm.Runtime_error(fmt.Sprintf("System ERROR: Get_argument() failed. err: %s", err.Error()))
	}
	return string(ret_b_sym)
}
func (vm *IcebergVM) itoentity(value interface{}) Entity {
	buf := new(bytes.Buffer)
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int64:
		err := binary.Write(buf, binary.LittleEndian, value.(int64))
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: itoentity() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_INT,
		}
	case reflect.Float64:
		err := binary.Write(buf, binary.LittleEndian, value.(float64))
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: itoentity() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_FLOAT,
		}
	case reflect.Bool:
		err := binary.Write(buf, binary.LittleEndian, value.(bool))
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: itoentity() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_BOOL,
		}
	case reflect.String:
		err := binary.Write(buf, binary.LittleEndian, []byte(value.(string)))
		if err != nil {
			vm.Runtime_error(fmt.Sprintf("System ERROR: itoentity() failed. err: %s", err.Error()))
		}
		return Entity{
			buf.Bytes(),
			T_STR,
		}
	default:
		vm.Runtime_error("VM ERROR: Tried to convert a value with type that is not compatible with Iceberg")
	}
	// It should not happen
	return Entity{}
}
func (vm *IcebergVM) Assign_var(symbol string, value interface{}) {
	source := vm.itoentity(value)
	registered, exist := vm.var_table[symbol]
	if exist {
		if registered.E_type != source.E_type {
			vm.Runtime_error("Type ERROR: Type mismatch")
		}
		vm.var_table[symbol] = source
	} else {
		test_ent := vm.conv_arg([]byte(symbol))
		if test_ent.E_type != T_UNDET {
			vm.Runtime_error(fmt.Sprintf("Type ERROR: Invalid symbol name %s", symbol))
		}
		vm.var_table[symbol] = source
	}
}

func (vm *IcebergVM) Dump_bytecode(code Bytecode) {
	for i, instr := range code.inst_list {
		fmt.Printf("%d: %s ", i, instr.Inst)
		for _, arg := range instr.Args {
			fmt.Printf("%x<type: %d>, ", arg.Data, arg.E_type)
		}
		fmt.Println("")
	}
	fmt.Println("Label table:")
	fmt.Println(code.label_table)
}

func (vm *IcebergVM) Run(code Bytecode) {
	vm.exec_pos = 0
	vm.label_table = code.label_table
	inst_max := int64(len(code.inst_list) - 1)

	for ;vm.exec_pos<=inst_max; {
		instr := code.inst_list[vm.exec_pos]
		vm.Inst_table[instr.Inst].Function(instr.Args)
		vm.exec_pos++
	}
}

func (vm *IcebergVM) inst_nop(args []Entity) {
	
}

func (vm *IcebergVM) inst_let(args []Entity) {
	sym_name := vm.Get_baresymbol(args[0])
	value, _ := vm.Get_argument(args[1], T_ANY)
	vm.Assign_var(sym_name, value)
}

func (vm *IcebergVM) arb_calc(args []Entity, operator string) {
	ope_a, type_a := vm.Get_argument(args[0], T_INT | T_FLOAT)
	ope_b, _ := vm.Get_argument(args[1], type_a)

	var ope_a_s, ope_b_s float64
	if type_a == T_INT {
		ope_a_s = float64(ope_a.(int64))
		ope_b_s = float64(ope_b.(int64))
	} else {
		ope_a_s = ope_a.(float64)
		ope_b_s = ope_b.(float64)
	}

	var is_divr bool
	var source float64
	switch operator {
	case "+":
		source = ope_a_s + ope_b_s
	case "-":
		source = ope_a_s - ope_b_s
	case "*":
		source = ope_a_s * ope_b_s
	case "//":
		if ope_b_s == 0 {
			vm.Runtime_error("Math ERROR: Division by zero")
		}
		source = math.Floor(ope_a_s / ope_b_s)
	case "/":
		if ope_b_s == 0 {
			vm.Runtime_error("Math ERROR: Division by zero")
		}
		source = ope_a_s / ope_b_s
		is_divr = true
	case "%":
		if ope_b_s == 0 {
			vm.Runtime_error("Math ERROR: Division by zero")
		}
		source = float64(int64(ope_a_s) % int64(ope_b_s))
	case "**":
		source = math.Pow(ope_a_s, ope_b_s)
	default:
		vm.Runtime_error(fmt.Sprintf("System ERROR: Unknown operator %s Possibly a bug in VM", operator))
	}

	var ans interface{}
	if type_a == T_INT && !is_divr {
		ans = int64(source)
	} else {
		ans = source
	}

	sym_name := vm.Get_baresymbol(args[2])
	vm.Assign_var(sym_name, ans)
}

func (vm *IcebergVM) inst_add(args []Entity) {
	vm.arb_calc(args, "+")
}
func (vm *IcebergVM) inst_sub(args []Entity) {
	vm.arb_calc(args, "-")
}
func (vm *IcebergVM) inst_mul(args []Entity) {
	vm.arb_calc(args, "*")
}
func (vm *IcebergVM) inst_div(args []Entity) {
	vm.arb_calc(args, "//")
}
func (vm *IcebergVM) inst_div_r(args []Entity) {
	vm.arb_calc(args, "/")
}
func (vm *IcebergVM) inst_mod(args []Entity) {
	vm.arb_calc(args, "%")
}
func (vm *IcebergVM) inst_pow(args []Entity) {
	vm.arb_calc(args, "**")
}

func (vm *IcebergVM) inst_cmp(args []Entity) {
	ope_a, type_a := vm.Get_argument(args[0], T_ANY ^ T_BOOL ^ T_LABEL)
	ope_b, _ := vm.Get_argument(args[1], T_STR)
	ope_c, _ := vm.Get_argument(args[2], type_a)

	var source bool
	if type_a == T_STR {
		switch ope_b.(string) {
		case ">":
			source = ope_a.(string) > ope_c.(string)
		case ">=":
			source = ope_a.(string) >= ope_c.(string)
		case "==":
			source = ope_a.(string) == ope_c.(string)
		case "<=":
			source = ope_a.(string) <= ope_c.(string)
		case "<":
			source = ope_a.(string) < ope_c.(string)
		case "!=":
			source = ope_a.(string) != ope_c.(string)
		default:
			vm.Runtime_error(fmt.Sprintf("Argument ERROR: Unknown oeprator %s", ope_b.(string)))
		}
	} else {
		var ope_a_s, ope_c_s float64
		if type_a == T_INT {
			ope_a_s = float64(ope_a.(int64))
			ope_c_s = float64(ope_c.(int64))
		} else {
			ope_a_s = ope_a.(float64)
			ope_c_s = ope_c.(float64)
		}
		switch ope_b.(string) {
		case ">":
			source = ope_a_s > ope_c_s
		case ">=":
			source = ope_a_s >= ope_c_s
		case "==":
			source = ope_a_s == ope_c_s
		case "<=":
			source = ope_a_s <= ope_c_s
		case "<":
			source = ope_a_s < ope_c_s
		case "!=":
			source = ope_a_s != ope_c_s
		default:
			vm.Runtime_error(fmt.Sprintf("Argument ERROR: Unknown oeprator %x, %x", ope_b.(string), "<"))
		}
	}
	sym_name := vm.Get_baresymbol(args[3])
	vm.Assign_var(sym_name, source)
}
func (vm *IcebergVM) arb_bool(args []Entity, operator string) {
	ope_a, _ := vm.Get_argument(args[0], T_BOOL)
	ope_b, _ := vm.Get_argument(args[1], T_BOOL)

	var source bool
	switch operator {
	case "and":
		source = ope_a.(bool) && ope_b.(bool)
	case "or":
		source = ope_a.(bool) || ope_b.(bool)
	case "xor":
		source = ope_a.(bool) != ope_b.(bool)
	default:
		vm.Runtime_error(fmt.Sprintf("System ERROR: Unknown operator %s Possibly a bug in VM", operator))
	}
	sym_name := vm.Get_baresymbol(args[2])
	vm.Assign_var(sym_name, source)
}
func (vm *IcebergVM) inst_and(args []Entity) {
	vm.arb_bool(args, "and")
}
func (vm *IcebergVM) inst_or(args []Entity) {
	vm.arb_bool(args, "or")
}
func (vm *IcebergVM) inst_xor(args []Entity) {
	vm.arb_bool(args, "xor")
}
func (vm *IcebergVM) inst_not(args []Entity) {
	operand, _ := vm.Get_argument(args[0], T_BOOL)
	sym_name := vm.Get_baresymbol(args[1])
	vm.Assign_var(sym_name, !operand.(bool))
}

func (vm *IcebergVM) inst_int(args []Entity) {
	operand, type_o := vm.Get_argument(args[1], T_INT | T_FLOAT)

	var source int64
	if type_o == T_INT {
		vm.Runtime_warning("Unnecessary cast T_INT->T_INT")
		source = operand.(int64)
	} else {
		source = int64(operand.(float64))
	}
	sym_name := vm.Get_baresymbol(args[0])
	vm.Assign_var(sym_name, source)
}
func (vm *IcebergVM) inst_float(args []Entity) {
	operand, type_o := vm.Get_argument(args[1], T_INT | T_FLOAT)

	var source float64
	if type_o == T_FLOAT {
		vm.Runtime_warning("Unnecessary cast T_FLOAT->T_FLOAT")
		source = operand.(float64)
	} else {
		source = float64(operand.(int64))
	}
	sym_name := vm.Get_baresymbol(args[0])
	vm.Assign_var(sym_name, source)
}
func (vm *IcebergVM) inst_bool(args []Entity) {
	operand, type_o := vm.Get_argument(args[1], T_ANY ^ T_LABEL)

	var source bool
	switch type_o {
	case T_INT:
		source = operand.(int64) != 0
	case T_FLOAT:
		source = operand.(float64) != 0.0
	case T_BOOL:
		vm.Runtime_warning("Unnecessary cast T_BOOL->T_BOOL")
		source = operand.(bool)
	case T_STR:
		source = operand.(string) != ""
	}
	sym_name := vm.Get_baresymbol(args[0])
	vm.Assign_var(sym_name, source)
}
func (vm *IcebergVM) inst_str(args []Entity) {
	operand, type_o := vm.Get_argument(args[1], T_ANY ^ T_LABEL)

	var source string
	switch type_o {
	case T_INT:
		source = strconv.FormatInt(operand.(int64), 10)
	case T_FLOAT:
		source = strconv.FormatFloat(operand.(float64), 'f', -1, 64)
	case T_BOOL:
		if operand.(bool) {
			source = "false"
		} else {
			source = "true"
		}
	case T_STR:
		vm.Runtime_warning("Unnecessary cast T_STR->T_STR")
		source = operand.(string)
	}
	sym_name := vm.Get_baresymbol(args[0])
	vm.Assign_var(sym_name, source)
}

func (vm *IcebergVM) inst_cat(args []Entity) {
	ope_a, _ := vm.Get_argument(args[0], T_STR)
	ope_b, _ := vm.Get_argument(args[1], T_STR)

	sym_name := vm.Get_baresymbol(args[2])
	vm.Assign_var(sym_name, ope_a.(string) + ope_b.(string))
}

func (vm *IcebergVM) inst_goto(args []Entity) {
	operand, _ := vm.Get_argument(args[0], T_LABEL)

	prog_idx, exist := vm.label_table[operand.(string)]
	if !exist {
		vm.Runtime_error(fmt.Sprintf("Argument ERROR: Unset label %s", operand.(string)))
	}
	vm.exec_pos = prog_idx
}
func (vm *IcebergVM) inst_when(args []Entity) {
	operand, _ := vm.Get_argument(args[1], T_LABEL)
	criteria, _ := vm.Get_argument(args[0], T_BOOL)

	prog_idx, exist := vm.label_table[operand.(string)]
	if !exist {
		vm.Runtime_error(fmt.Sprintf("Argument ERROR: Unset label %s", operand.(string)))
	}
	if criteria.(bool) {
		vm.exec_pos = prog_idx
	}
}

func (vm *IcebergVM) inst_dump(args []Entity) {
	fmt.Println("Dump begin ---")
	fmt.Println("Variable Symbol Table:")
	for key, value := range vm.var_table {
		cnv, c_type := vm.Get_argument(value, T_ANY)
		fmt.Printf("%s -> %v <type: %d>\n", key, cnv, c_type)
	}
	fmt.Println("Dump end---")
}

//func (vm *IcebergVM) inst_print(args []Entity) {
//	operand, _ := vm.Get_argument(args[0], T_STR)
//	fmt.Println(operand.(string))
//}

func (vm *IcebergVM) Init() {
	vm.Inst_table = make(map[string]InstructionDesc)
	vm.label_table = make(map[string]int64)
	vm.var_table = make(map[string]Entity)
	
	vm.Inst_table["nop"] = InstructionDesc{ vm.inst_nop, 0, }
	vm.Inst_table["let"] = InstructionDesc{ vm.inst_let, 2, }
	vm.Inst_table["add"] = InstructionDesc{ vm.inst_add, 3, }
	vm.Inst_table["sub"] = InstructionDesc{ vm.inst_sub, 3, }
	vm.Inst_table["mul"] = InstructionDesc{ vm.inst_mul, 3, }
	vm.Inst_table["div"] = InstructionDesc{ vm.inst_div, 3, }
	vm.Inst_table["div_r"] = InstructionDesc{ vm.inst_div_r, 3, }
	vm.Inst_table["mod"] = InstructionDesc{ vm.inst_mod, 3, }
	vm.Inst_table["pow"] = InstructionDesc{ vm.inst_pow, 3, }
	vm.Inst_table["cmp"] = InstructionDesc{ vm.inst_cmp, 4, }
	vm.Inst_table["and"] = InstructionDesc{ vm.inst_and, 3, }
	vm.Inst_table["or"] = InstructionDesc{ vm.inst_or, 3, }
	vm.Inst_table["xor"] = InstructionDesc{ vm.inst_xor, 3, }
	vm.Inst_table["not"] = InstructionDesc{ vm.inst_not, 2, }
	vm.Inst_table["int"] = InstructionDesc{ vm.inst_int, 2, }
	vm.Inst_table["float"] = InstructionDesc{ vm.inst_float, 2, }
	vm.Inst_table["bool"] = InstructionDesc{ vm.inst_bool, 2, }
	vm.Inst_table["str"] = InstructionDesc{ vm.inst_str, 2, }
	vm.Inst_table["cat"] = InstructionDesc{ vm.inst_cat, 3, }
	vm.Inst_table["goto"] = InstructionDesc{ vm.inst_goto, 1, }
	vm.Inst_table["when"] = InstructionDesc{ vm.inst_when, 2, }

	vm.Inst_table["dump"] = InstructionDesc{ vm.inst_dump, 0, }
	
	//vm.Inst_table["print"] = InstructionDesc{ vm.inst_print, 1, }
}
