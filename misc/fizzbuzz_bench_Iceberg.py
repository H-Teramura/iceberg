import iceberg
import time

class testVM(iceberg.IcebergVM):
    def inst_print(self, args):
        operand = self.get_argument(args[0], iceberg.T_STR)
        print(operand)
        
    def add_other_symbols(self):
        self.const_opcode['print'] = iceberg.InstructionDesc(self.inst_print, 1)

vm = testVM()

with open('main.ib') as f:
    src = f.read()

print("Compiling...")

c_time_s = time.time()
bytecode = vm.gen_bytecode(src)
c_time_d = time.time() - c_time_s

print("Running FizzBuzz...")

e_time_s = time.time()
vm.run(bytecode)
e_time_d = time.time() - e_time_s

print("Result:")
print("Compilation time", c_time_d, "sec.")
print("Execution time", e_time_d, "sec.")
