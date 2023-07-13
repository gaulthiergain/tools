SECTIONS
{
 . = 0x100000;
 _text = .;
 .text :
 {
  KEEP (*(.data.boot))
  *(.text.boot)
  *(.text)
  *(.text.*) /* uncomment it to dissagregate functions */
 }

<END_TEXT_REPLACE_LOCATION>
 _etext = .;
 . = ALIGN((1 << 12)); __eh_frame_start = .; .eh_frame : { *(.eh_frame) *(.eh_frame.*) } __eh_frame_end = .; __eh_frame_hdr_start = .; .eh_frame_hdr : { *(.eh_frame_hdr) *(.eh_frame_hdr.*) } __eh_frame_hdr_end = .;
 . = ALIGN((1 << 12)); uk_ctortab_start = .; .uk_ctortab : { KEEP(*(SORT_BY_NAME(.uk_ctortab[0-9]))) } uk_ctortab_end = .;
<UK_INIT_REPLACE_LOCATION> uk_inittab_start = .; .uk_inittab : { KEEP(*(SORT_BY_NAME(.uk_inittab[1-6][0-9]))) } uk_inittab_end = .;

<RODATA_REPLACE_LOCATION>
 . = ALIGN((1 << 12));
  _rodata = .;
 .rodata :
 {
<INNER_RODATA>
   *(.rodata)
   *(.rodata.*)
 }

<ERODATA_REPLACE_LOCATION>
 _erodata = .;
 . = ALIGN(0x8);
 _ctors = .;
 .preinit_array : {
  PROVIDE_HIDDEN (__preinit_array_start = .);
  KEEP (*(.preinit_array))
  PROVIDE_HIDDEN (__preinit_array_end = .);
 }
 . = ALIGN(0x8);
 .init_array : {
  PROVIDE_HIDDEN (__init_array_start = .);
  KEEP (*(SORT_BY_INIT_PRIORITY(.init_array.*) SORT_BY_INIT_PRIORITY(.ctors.*)))
  KEEP (*(.init_array .ctors))
  PROVIDE_HIDDEN (__init_array_end = .);
 }
 _ectors = .;
 . = ALIGN(0x8); _tls_start = .; .tdata : { *(.tdata) *(.tdata.*) *(.gnu.linkonce.td.*) } _etdata = .;

<DATA_REPLACE_LOCATION>
 . = ALIGN((1 << 12));
 _data = .;
 .data :
 {
<INNER_DATA>
   *(.data)
   *(.data.*)
 }

<EDATA_REPLACE_LOCATION>
 _edata = .;
 . = ALIGN((1 << 12));

<BSS_REPLACE_LOCATION>
 __bss_start = .;
 .bss :
 {
<INNER_BSS>
   *(.bss)
   *(.bss.*)
  *(COMMON)
  . = ALIGN((1 << 12));
 }

<TBSS_REPLACE_LOCATION>
 .tbss : { *(.tbss) *(.tbss.*) *(.gnu.linkonce.tb.*) . = ALIGN(0x8); } _tls_end = . + SIZEOF(.tbss);

<INTRSTACK_REPLACE_LOCATION>
 .intrstack :
 {
  *(.intrstack)
  . = ALIGN((1 << 12));
 }
 _end = .;
 .comment 0 : { *(.comment) }
 .debug 0 : { *(.debug) } .line 0 : { *(.line) } .debug_srcinfo 0 : { *(.debug_srcinfo) } .debug_sfnames 0 : { *(.debug_sfnames) } .debug_aranges 0 : { *(.debug_aranges) } .debug_pubnames 0 : { *(.debug_pubnames) } .debug_info 0 : { *(.debug_info .gnu.linkonce.wi.*) } .debug_abbrev 0 : { *(.debug_abbrev) } .debug_line 0 : { *(.debug_line .debug_line.* .debug_line_end ) } .debug_frame 0 : { *(.debug_frame) } .debug_str 0 : { *(.debug_str) } .debug_loc 0 : { *(.debug_loc) } .debug_macinfo 0 : { *(.debug_macinfo) } .debug_weaknames 0 : { *(.debug_weaknames) } .debug_funcnames 0 : { *(.debug_funcnames) } .debug_typenames 0 : { *(.debug_typenames) } .debug_varnames 0 : { *(.debug_varnames) } .debug_pubtypes 0 : { *(.debug_pubtypes) } .debug_ranges 0 : { *(.debug_ranges) } .debug_macro 0 : { *(.debug_macro) } .gnu.attributes 0 : { KEEP (*(.gnu.attributes)) }
 /DISCARD/ : { *(.note.gnu.build-id) }
}