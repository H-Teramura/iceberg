#!/usr/bin/env python
# -*- coding:utf-8 -*-
"""The Iceberg script language implementation"""
# Constants
# -Types
T_INT   = 1
"""type flag for int type in Iceberg"""
T_FLOAT = 2
"""type flag for float type in Iceberg"""
T_BOOL  = 4
"""type flag for bool type in Iceberg"""
T_STR   = 8
"""type flag for str type in Iceberg"""
T_LABEL = 16
"""type flag for label type in Iceberg"""

T_ANY   = 31
"""type flag that represents any type in Iceberg

useful if you do not want specific types in get_argument function"""

class Instruction:
    """data structure for storing a opcode and a list of arguments."""
    def __init__(self, opcode, args):
        """constructor

        opcode: instruction in str
        args: list of arguments
        returns new instance"""
        self.opcode = opcode
        self.args = args

    def __str__(self):
        return self.opcode + str(self.args)

class InstructionDesc:
    """data structure for storing instruction function and number of arguments needed."""
    def __init__(self, func, n_args):
        """constructor

        func: function that processes corresponding instruction. The function should be a method of IcebergVM or its children and take a list of arguments as its second argument
        n_args: number of arguments required for the instruction
        returns new instance"""
        self.func = func
        self.n_args = n_args

    def execute(self, args):
        """call a function that processes corresponding instruction
        args: list of arguments
        returns None"""
        self.func(args)
    
class Bytecode:
    """data structure for storing compiled instruction list and label table"""
    def __init__(self, instructions, label_table):
        """constructor

        instructions: list of Instruction class that represents a program
        label_table: dictionary that has labels as keys and stores where the labels are pointing
        returns new instance"""
        self.instructions = instructions
        self.label_table = label_table
    
class IcebergVM:
    """base class for Iceberg virtual machine.

    users may inherit this class to add extra functionalities such as additional instructions or variables.
    """
    def add_other_symbols(self):
        """dummy function for adding functionality to the VM.

        users can override this function to add instructions, make new symbols on variable table(dict var_table) or etc.
        the first argument 'self' should be the only argument
        returns None"""
        pass

    def raise_error(self, message):
        """print a error on compile and causes RuntimeError

        message: error message in str
        raises RuntimeError"""
        # On compile, exec_pos stores the index of lines being compiled
        # so the bad line number is [exec_pos+1]
        print("In line " + str(self.exec_pos + 1) + ",")
        print(message)
        raise RuntimeError

    def raise_runtime_error(self, message):
        """print a error on runtime and causes RuntimeError

        message: error message in str
        raises RuntimeError"""
        print("Iceberg Runtime ERROR!")
        print("In instruction number " + str(self.exec_pos) + ",")
        print(message)
        raise RuntimeError

    def raise_runtime_warning(self, message):
        """print a warning on runtime and causes RuntimeError

        a warning tells users a suspicious instruction that might be caused by bug(s) in the script
        message: warning message in str
        returns None"""
        print("WARNING: In instruction number " + str(self.exec_pos) + ",")
        print(message)

    def chk_nargs(self, args, expected_n_args):
        """check if number of given arguments matches required one

        if the number does not match, the function calls self.raise_error() causing RuntimeError
        args: list of arguments
        expected_n_args: number of arguments expected
        returns None"""
        n_args = len(args)
        if n_args < expected_n_args:
            self.raise_error("Argument ERROR: Too few arguments (expected " + str(expected_n_args) + " but " +  str(n_args) + " given)")
        elif n_args > expected_n_args:
            self.raise_error("Argument ERROR: Too many arguments (expected " + str(expected_n_args) + " but " +  str(n_args) + " given)")
            
    def parse_args(self, argstr):
        """convert argument part of script into list of arguments

        if there is a error in syntax, the function calls self.raise_error()
        argstr: argument part of one-line script
        returns list of arguments"""
        args = []
        buf = []

        # true if processing between a pair of quotation marks
        d_quote = False
        s_quote = False

        # true if processing between ending brackets(secoond quotes or ])
        after_brackets = False
        for c in argstr:
            if d_quote == True:
                if c == '"':
                    # end of quotation block
                    args.append('"' + ''.join(buf) + '"')
                    buf = []
                    d_quote = False
                    after_brackets = True
                else:
                    buf.append(c)
            elif s_quote == True:
                if c == "'":
                    #end of quotation block
                    args.append("'" + "".join(buf) + "'")
                    buf = []
                    s_quote = False
                    after_brackets = True
                else:
                    buf.append(c)
            else:
                if c == ",":
                    if not after_brackets:
                        args.append(''.join(buf))
                        buf = []
                    else:
                        # ignore values between ending brackets and comma(TODO: such values should be treated as syntax errors)
                        after_brackets = False
                elif c == '"':
                    d_quote = True
                elif c == "'":
                    s_quote = True
                elif c != " ":
                    buf.append(c)

        if not len(buf) == 0:
            # if buf has something at the end of argstr, it is the last argument
            args.append(''.join(buf))

        # if d_quote or s_quote are still true, something is wrong with the script
        if d_quote:
            self.raise_error('Syntax ERROR: Missing "')
        if s_quote:
            self.raise_error("Syntax ERROR: Missing '")

        return args
    
    def parse_oneline(self, line, list_script):
        """convert a line of script into Instruction class

        if there is a error in syntax, the function calls self.raise_error()
        line: a line of script
        list_script: list of Instructions to append the line
        returns None"""
        if not ' ' in line:
            # must be a instruction with no argument
            instr = line
            if instr in self.const_opcode:
                # valid instruction
                # if number of arguments is wrong compilation fails here
                self.chk_nargs([], self.const_opcode[instr].n_args)
                # add the instruction and its argument(that is blank list) to the list of Instructions
                list_script.append(Instruction(instr, []))
            elif instr.find('@') == 0:
                # if it was a label, simply add it to list(will be processed in function set_label
                list_script.append(Instruction(instr, []))
            elif instr == '':
                # blank line
                pass
            else:
                # when strange thing is written
                self.raise_error("Syntax ERROR: Unknown instruction: " + instr)
        else:
            # if there are some spaces, split line with the first space
            line_list = line.split(' ', maxsplit=1)
            # first thing must be a instruction
            instr = line_list[0]
            if instr in self.const_opcode:
                # it is valid instruction
                # second thing must be a series of arguments
                args = self.parse_args(line_list[1])
                # same as above
                self.chk_nargs(args, self.const_opcode[instr].n_args)
                list_script.append(Instruction(instr, args))
            elif instr.find('@') == 0:
                # all labels should not have something after it
                self.raise_error("Syntax ERROR: Expected newline after label definition")
            else:
                self.raise_error("Syntax ERROR: Unknown instruction: " + instr)

    def set_label(self, list_script):
        """create label table out of list of Instructions

        list_script: list of Instructions
        returns dictionary that has labels as keys and position of labels as values"""
        label_table = {}
        for i,inst in enumerate(list_script):
            if inst.opcode.find('@') == 0:
                label_table[inst.opcode] = i
                # Trivia: Actually, labels are treated as nop instructions. In branch instructions such as when or goto, label table is looked up and exec_pos is changed with the value of the label table.
                inst.opcode = 'nop'

        return label_table
                
    def parse_script(self, script):
        """convert a script into a set of list of instructions and a label table.

        script: script in str
        returns tuple (list_of_Instructions, label_table_dict)"""
        inst_number = 0
        list_script = []
        # newlines are delimiters for Iceberg scripts to break them into lines 
        lines = script.split("\n")
        for i,line in enumerate(lines):
            # exec_pos is current_line_number - 1
            self.exec_pos = i
            # ignore tabs or spaces on the left sides of lines
            line = line.lstrip()
            # interpret a line
            self.parse_oneline(line, list_script)
            
        return (list_script, self.set_label(list_script))
    
    def gen_bytecode(self, script):
        """compile a script into Bytecode class.

        script: script in str
        returns Bytecode class"""
        (list_script, label_table) = self.parse_script(script)
        return Bytecode(list_script, label_table)
    
    def run(self, bytecode):
        """execute a bytecode.

        bytecode: Bytecode to execute
        returns None"""
        # exec_pos here is DIFFERENT fron that is in function parse_script. in this function, exec_pos indicates the index of instructions not the script line index!
        self.exec_pos = 0
        self.label_table = bytecode.label_table
        inst_max = len(bytecode.instructions) - 1
        # using while loop because branch instructions can be inplemented rather easily
        while self.exec_pos <= inst_max:
            inst = bytecode.instructions[self.exec_pos]
            self.const_opcode[inst.opcode].execute(inst.args)
            self.exec_pos += 1

    def get_type_flag(self, value):
        """return type flag of value

        value: value to get a flag of
        returns type flag in int"""
        if type(value) == int:
            return T_INT
        elif type(value) == float:
            return T_FLOAT
        elif type(value) == bool:
            return T_BOOL
        elif type(value) == str:
            return T_STR
        else:
            self.raise_runtime_error("System ERROR: Argument "  + str(value) + " is unknown type")

    def get_argument(self, arg, type_flag):
        """convert argument to pythonic value.

        the type flag of the argument should be included in type_flag or the function calls self.raise_runtime_error()
        also, if arg was symbol and this is not in symbol tables(const_table, var_table), the function calls self.raise_runtime_error()
        arg: argument in str
        type_flag: a type_flag or OR-ed set of flags(such as T_INT, T_ANY^T_LABEL)
        returns argument in appropreate pythonic type(for example, if arg is '"hello"'(that is T_STR in Iceberg), return value will be 'hello')"""
        # Defined symbol?
        if arg in self.const_table:
            if type_flag & self.get_type_flag(self.const_table[arg]):
                return self.const_table[arg]
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
        if arg in self.var_table:
            if type_flag & self.get_type_flag(self.var_table[arg]):
                return self.var_table[arg]
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")

        # Label?
        if arg.find('@') == 0:
            if type_flag & T_LABEL:
                return arg
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
                
        # String literal?
        if arg.find('"') == 0:
            if type_flag & T_STR:
                return arg.strip('"')
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
        if arg.find("'") == 0:
            if type_flag & T_STR:
                return arg.strip("'")
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")

        # Boolean?
        if arg == "true":
            if type_flag & T_BOOL:
                return True
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
        if arg == "false":
            if type_flag & T_BOOL:
                return False
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")

        # FPN?
        if arg.find('.') != -1:
            try:
                f_value = float(arg)
                if type_flag & T_FLOAT:
                    return f_value
                else:
                    self.raise_runtime_error("Type ERROR: Type mismatch")
            except ValueError:
                # arg can be unbound symbol with '.'
                pass

        # Integer?
        try:
            i_value = int(arg)
            if type_flag & T_INT:
                return i_value
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
        except ValueError:
            # Unbound symbol
            self.raise_runtime_error("Argument ERROR: Unbound symbol " + arg)
        
    def assign_var(self, symbol, value):
        """assign value to symbol.

        if specified symbol exists, the type of the symbol is checked. if it does not match the type of value, self.raise_runtime_error() will be called instead of assigning
        if the symbol does not exist, new symbol entry will be created in var_table.
        the symbol cannot be constant such as number, and start with @ or quotation marks
        symbol: symbol in str
        value: value to assign
        returns None"""
        # is symbol already assigned?
        if symbol in self.var_table:
            # update value if the type matches(you cannot alter type!)
            if type(self.var_table[symbol]) == type(value):
                self.var_table[symbol] = value
            else:
                self.raise_runtime_error("Type ERROR: Type mismatch")
        else:
            # if unbound, check if the given symbol is appropreate for a symbol
            try:
                temp = float(symbol)
                # if the symbol looks like a number...
                self.raise_runtime_error("Type ERROR: Cannot assign value to constant")
            except ValueError:
                if symbol.find("'") == 0 or symbol.find('"') == 0:
                    # symbols cannot start with quotations
                    self.raise_runtime_error("Type ERROR: Cannot assign value to constant")
                if symbol.find('@') == 0:
                    # or @ (Used for labels)
                    self.raise_runtime_error("Type ERROR: Cannot assign value to label")

                # given symbol name is valid!
                self.var_table[symbol] = value
    
    # instruction definitions
    def inst_nop(self, args):
        """function for instruction 'nop'

        the actual behavior of every instructions is determined by functions like inst_nop(self, args)(this function) or inst_let(self, args). the function should be method of IcebergVM or its children and take list of arguments as the second argument. Its return value will be ignored. inst_when function is a good example and very useful for writing your own extensions
        below is general information of the functions
        args: list of arguments
        returns None. raising exceptions inside the function is ok"""
        pass
    def inst_let(self, args):
        self.assign_var(args[0], self.get_argument(args[1], T_ANY))
        
    # Arithmetric instruction helper
    # made to avoid duplicates in source code
    def arb_calc(self, args, operator):
        ope_a = self.get_argument(args[0], T_INT | T_FLOAT)
        # ope_a and ope_b should be the same type
        ope_b = self.get_argument(args[1], self.get_type_flag(ope_a))
        
        if operator == '+':
            source = ope_a + ope_b
        elif operator == '-':
            source = ope_a - ope_b
        elif operator == '*':
            source = ope_a * ope_b
        elif operator == '//':
            if ope_b != 0:
                source = ope_a // ope_b
            else:
                self.raise_runtime_error("Math ERROR: Division by zero")
        elif operator == '/':
            if ope_b != 0:
                source = ope_a / ope_b
            else:
                self.raise_runtime_error("Math ERROR: Division by zero")
        elif operator == '%':
            if ope_b != 0:
                source = ope_a % ope_b
            else:
                self.raise_runtime_error("Math ERROR: Division by zero")
        elif operator == '**':
            source = ope_a ** ope_b
        else:
            self.raise_runtime_error("System ERROR: Unknown operator " + str(operator) + " Possibly a bug in VM")

        self.assign_var(args[2], source)
        
    def inst_add(self, args):
        self.arb_calc(args, '+')
        
    def inst_sub(self, args):
        self.arb_calc(args, '-')
        
    def inst_mul(self, args):
        self.arb_calc(args, '*')
        
    def inst_div(self, args):
        self.arb_calc(args, '//')
        
    def inst_div_r(self, args):
        self.arb_calc(args, '/')
        
    def inst_mod(self, args):
        self.arb_calc(args, '%')
        
    def inst_pow(self, args):
        self.arb_calc(args, '**')
        
    def inst_cmp(self, args):
        ope_a = self.get_argument(args[0], T_ANY ^ T_BOOL ^ T_LABEL)
        ope_b = self.get_argument(args[1], T_STR)
        ope_c = self.get_argument(args[2], self.get_type_flag(ope_a))

        if ope_b == '==':
            self.assign_var(args[3], ope_a == ope_c)
        elif ope_b == '>':
            self.assign_var(args[3], ope_a > ope_c)
        elif ope_b == '>=':
            self.assign_var(args[3], ope_a >= ope_c)
        elif ope_b == '<=':
            self.assign_var(args[3], ope_a <= ope_c)
        elif ope_b == '<':
            self.assign_var(args[3], ope_a < ope_c)
        elif ope_b == '!=':
            self.assign_var(args[3], ope_a != ope_c)
        else:
            self.raise_runtime_error("Argument ERROR: Unknown operator " + ope_b)

    # Bool arithmetic helper
    def arb_bool(self, args, operator):
        ope_a = self.get_argument(args[0], T_BOOL)
        ope_b = self.get_argument(args[1], T_BOOL)
        
        if operator == 'and':
            source = ope_a and ope_b
        elif operator == 'or':
            source = ope_a or ope_b
        elif operator == 'xor':
            source = ope_a ^ ope_b
        else:
            self.raise_runtime_error("System ERROR: Unknown operator " + str(operator) + " Possibly a bug in VM")
        
        self.assign_var(args[2], source)

    def inst_and(self, args):
        self.arb_bool(args, 'and')
        
    def inst_or(self, args):
        self.arb_bool(args, 'or')
        
    def inst_xor(self, args):
        self.arb_bool(args, 'xor')
        
    def inst_not(self, args):
        operand = self.get_argument(args[0], T_BOOL)
        source = not operand
        self.assign_var(args[1], source)

    def inst_int(self, args):
        operand = self.get_argument(args[1], T_INT | T_FLOAT)
        if type(operand) == int:
            self.raise_runtime_warning("Casting to the same type")
        self.assign_var(args[0], int(operand))
        
    def inst_float(self, args):
        operand = self.get_argument(args[1], T_INT | T_FLOAT)
        if type(operand) == float:
            self.raise_runtime_warning("Casting to the same type")
        self.assign_var(args[0], float(operand))
        
    def inst_bool(self, args):
        operand = self.get_argument(args[1], T_ANY ^ T_LABEL)
        if type(operand) == bool:
            self.raise_runtime_warning("Casting to the same type")
        self.assign_var(args[0], bool(operand))
        
    def inst_str(self, args):
        operand = self.get_argument(args[1], T_ANY ^ T_LABEL)
        if type(operand) == str:
            self.raise_runtime_warning("Casting to the same type")
        self.assign_var(args[0], str(operand))

    def inst_cat(self, args):
        ope_a = self.get_argument(args[0], T_STR)
        ope_b = self.get_argument(args[1], T_STR)
        self.assign_var(args[2], ope_a + ope_b)
        
    def inst_goto(self, args):
        operand = self.get_argument(args[0], T_LABEL)
        if operand in self.label_table:
            # set exec_pos to the position of the label(ar runtime, nop instruction is stored).
            # the VM then next executes te instruction on [exec_pos+1] because exec_pos is always incremented after the executin of every instruction
            self.exec_pos = self.label_table[operand]
        else:
            # if the label does not exist
            self.raise_runtime_error("Argument ERROR: Unset label " + args[0])
        
    def inst_when(self, args):
        ope_a = self.get_argument(args[0], T_BOOL)
        ope_b = self.get_argument(args[1], T_LABEL)
        if ope_a:
            if ope_b in self.label_table:
                self.exec_pos = self.label_table[ope_b]
            else:
                self.raise_runtime_error("Argument ERROR: Unset label " + args[1])

    def inst_dump(self, args):
        print("Dump begin ---")
        print("Constant symbol table")
        print(self.const_table)
        print()
        print("Variable symbol table")
        print(self.var_table)
        print()
        print("Label table")
        print(self.label_table)
        print("Dump end ---")
                    
    def __init__(self):
        """constructor

        creates Iceberg virtual machine and compiler.
        returns new instance"""
        self.exec_pos = 0
        self.label_table = {}
        self.var_table = {}
        self.const_table = {}
        self.const_opcode = {}

        # register instructions. this is necessary for instructions to be recognized by the VM
        self.const_opcode['nop'] = InstructionDesc(self.inst_nop, 0)
        self.const_opcode['let'] = InstructionDesc(self.inst_let, 2)
        self.const_opcode['add'] = InstructionDesc(self.inst_add, 3)
        self.const_opcode['sub'] = InstructionDesc(self.inst_sub, 3)
        self.const_opcode['mul'] = InstructionDesc(self.inst_mul, 3)
        self.const_opcode['div'] = InstructionDesc(self.inst_div, 3)
        self.const_opcode['div_r'] = InstructionDesc(self.inst_div_r, 3)
        self.const_opcode['mod'] = InstructionDesc(self.inst_mod, 3)
        self.const_opcode['pow'] = InstructionDesc(self.inst_pow, 3)
        self.const_opcode['cmp'] = InstructionDesc(self.inst_cmp, 4)
        self.const_opcode['and'] = InstructionDesc(self.inst_and, 3)
        self.const_opcode['or'] = InstructionDesc(self.inst_or, 3)
        self.const_opcode['xor'] = InstructionDesc(self.inst_xor, 3)
        self.const_opcode['not'] = InstructionDesc(self.inst_not, 2)
        self.const_opcode['int'] = InstructionDesc(self.inst_int, 2)
        self.const_opcode['float'] = InstructionDesc(self.inst_float, 2)
        self.const_opcode['bool'] = InstructionDesc(self.inst_bool, 2)
        self.const_opcode['str'] = InstructionDesc(self.inst_str, 2)
        self.const_opcode['cat'] = InstructionDesc(self.inst_cat, 3)
        self.const_opcode['goto'] = InstructionDesc(self.inst_goto, 1)
        self.const_opcode['when'] = InstructionDesc(self.inst_when, 2)

        self.const_opcode['dump'] = InstructionDesc(self.inst_dump, 0)
        
        self.add_other_symbols()
