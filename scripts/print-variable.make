#!/usr/bin/env -S make -f

# print-variable.make
#
# Print the value of a variable in a Makefile.
# This will execute some commands in the input Makefile, so be careful.
#
# Usage:
#   print-variable.make INPUT=/path/to/Makefile VARIABLE_NAME

% : ; @echo $($*)

include $(INPUT)
