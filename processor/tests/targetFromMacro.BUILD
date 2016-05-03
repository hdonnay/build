LIB_COMPILER_FLAGS = [
    "-std=c11",
    "-fasm",
    "-c",
    "-ffreestanding",
    "-fno-builtin",
    "-fno-omit-frame-pointer",
    "-fplan9-extensions",
    "-fvar-tracking",
    "-fvar-tracking-assignments",
    "-g",
    "-gdwarf-2",
    "-ggdb",
    "-mcmodel=small",
    "-mno-red-zone",
    "-O0",
    "-static",
    "-Wall",
    "-Wno-missing-braces",
    "-Wno-parentheses",
    "-Wno-unknown-pragmas"
]


harvey_library = cc_library(
	copts=LIB_COMPILER_FLAGS,
	includes=[
		"//sys/include",
		"//amd64/include",
	],
)

harvey_library(
	name="libString",
	srcs=[
		"string.c",
	]
)