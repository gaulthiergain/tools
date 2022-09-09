import re
import os
import hashlib
from collections import defaultdict
from binascii import hexlify

DIFF_FOLDER="diff" + os.sep
PAGE_FOLDER="pages" + os.sep
PAGE_VMA_FOLDER="pages_vma" + os.sep

DBG_EXTENSION = ".dbg"
PAGE_SIZE = 4096

SIZE  = "size"
ALIGN = "_local_align"
ASLR_PLT  = "_aslr_plt"
ASLR_DCE  = "_aslr_dce"
ASLR_DEFAULT  = "_aslr_default"

class Unikernel:
    def __init__(self, shortname, name):
        self.shortname = self.filter_name(shortname)
        self.name = name
        self.type_unikernel = self.process_name(shortname)
        self.segments = list()
        self.sections = list()
        self.byte_array = None
        self.map_addr_section = dict()
        self.map_symbols = defaultdict(list)
        self.dump = None
    
    def filter_name(self, name):
            return name.strip(DBG_EXTENSION)
    
    def process_name(self, shortname):
        if ALIGN in shortname:
            return ALIGN
        elif SIZE in shortname:
            return SIZE
        elif ASLR_PLT in shortname:
            return ASLR_PLT
        elif ASLR_DEFAULT in shortname:
            return ASLR_DEFAULT
        elif ASLR_DCE in shortname:
            return ASLR_DCE
        else:
            return ""

class Segment:
    def __init__(self, address, offset, size):
        self.address = address
        self.offset = offset
        self.size = size

class Section:
    def __init__(self, name, start, offset, size, alignment):
        self.name = name
        self.start = start
        self.start_align = self.round_mult()
        self.offset = offset
        self.size = size
        self.alignment = alignment
        self.end = start+size
        self.pages = list()
        self.data = None

    def round_mult(self, base=PAGE_SIZE):
        if self.start % PAGE_SIZE != 0:
            return base * round(self.start / base)
        return self.start

class Symbol:
    def __init__(self, address, name, info):
        self.address = address
        self.name = name
        self.info = info

class Page:
    def __init__(self, name, number, start, size, uk_name, sect_name, content):
        self.name = name
        self.number = number
        self.start = start
        self.size = size
        self.end = self.start+self.size
        self.uk_name = uk_name
        self.sect_name = sect_name
        self.content = self.process_content(content)
        self.instructions = list()
        self.instructions_string = ""
        self.others = defaultdict(list)
        self.zeroes = self.count_zeroes()
        self.hash = hashlib.sha256(self.content).hexdigest()

    def warning_symbol(self, map_symbols, addr):
        
        if len(map_symbols[addr]) > 1:
            #print("[WARNING] several symbols for {:02x}".format(addr))
            for t in map_symbols[addr]:
                #print("\t-{} {} 0x{:02x}".format(t.name, t.info, t.address))
                if t.name not in self.others[t.address]:
                    self.others[t.address].append(t.name)
            return " + " + str(len(map_symbols[addr])) + "others "
        return ""

    def disassemble_bytes(self):

        str_ascii = ""
        for i, b in enumerate(self.content):

            if i == 0:
                self.instructions_string +="0x{:02x}".format(self.start) + ": "
            if i > 0 and i % 4 == 0:
                self.instructions_string += " "
            if i > 0 and i % 16 == 0:
                self.instructions_string += str_ascii
                self.instructions_string += "\n" + "0x{:02x}".format(self.start + i) + ": "
                str_ascii = ""

            fb = "{:02x}".format(b)
            self.instructions_string += fb
            if int(fb, 16) > 20 and int(fb, 16) < 126:
                    ascii_str = "%s" % bytearray.fromhex(fb).decode()
            else:
                    ascii_str = "."
            str_ascii += ascii_str

    def instructions_to_string(self, map_symbols):
        
        for ins in self.instructions:
            function_call = ""

            # FUNCTION NAME
            if ins.address in map_symbols:
                ret = self.warning_symbol(map_symbols, ins.address)
                self.instructions_string += "\n[== " + map_symbols[ins.address][0].name + ret + " ==]\n"

            #FUNCTION CALL
            regex = r"0x[a-f0-9]*"
            matches = re.finditer(regex, ins.op_str)
            for _, z in enumerate(matches, start=1):
                addr = int(z.group(),16)
                if addr in map_symbols:
                    ret = self.warning_symbol(map_symbols, addr)
                    function_call = "(" + map_symbols[addr][0].name + ret + ")"

            self.instructions_string += "{: <32} 0x{:02x} {: <10} {: <10}{}\n".format(ins.bytes, ins.address, ins.mnemonic, ins.op_str, function_call)

        for k,values in self.others.items():
            self.instructions_string += "\n0x{:02x}: [".format(k) 
            for v in values:
                self.instructions_string += "" + v + ","
            self.instructions_string +=("]")

    def process_content(self, content):

        # ALIGN if necessary
        if len(content) % PAGE_SIZE != 0:
            byte_array = bytearray([0] * PAGE_SIZE)
            byte_array[0:len(content)] = content
            return byte_array

        return content

    def count_zeroes(self):
        zeroes = 0
        for c in self.content:
            if c == 0:
                zeroes += 1
        return zeroes

class Instruction:
    def __init__(self, address, mnemonic, op_str, _bytes):
        self.address = address
        self.mnemonic = mnemonic
        self.op_str = op_str
        self.bytes = self.cut(hexlify(_bytes).decode())
    
    def cut(self, line, n=2):
        return ' '.join([line[i:i+n] for i in range(0, len(line), n)])

class Dump:
    def __init__(self, shortname, name, content):
        self.shortname = shortname
        self.name = name
        self.content = content
        self.pages = list()