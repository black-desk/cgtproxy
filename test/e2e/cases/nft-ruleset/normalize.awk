# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Keep nft text and rule order. Normalize counter values, the root cgroup ID,
# and the presentation order of independent objects and set/map elements.
# Ignore the owner's process-name comment, but retain the table flags.
# Rule/expression order and counter presence and position remain significant.
function emit(line) { block = block line "\n" }
/^#/ { next }
/^table / { sub(/ # progname .*/, ""); header = $0; next }
/^\tflags / && key == "" { table_flags = $0; next }
/^\t(set|map|chain) / { key = $1 "/" $2; object = $2; block = "" }
/^$/ || /^}$/ { next }
{
	gsub(/counter packets [0-9]+ bytes [0-9]+/, "counter packets 0 bytes 0")
	if ($0 ~ /^[\t ]*elements = \{/) {
		elements = $0
		while (elements !~ /}/) {
			if ((getline) <= 0) exit 1
			elements = elements " " $0
		}
		sub(/^[^{]*\{ */, "", elements)
		sub(/ *}.*$/, "", elements)
		n = split(elements, values, ",")
		for (i = 1; i <= n; i++) {
			gsub(/^[\t ]+|[\t ]+$/, "", values[i])
			# TODO: Fix nft's cgroupv2_get_path() upstream to check the root
			# directory itself. nftables 1.0.9 searches only descendants and
			# therefore prints the root cgroup ID instead of "/".
			if (object == "cgroup-vmap")
				sub("^" root " :", "\"/\" :", values[i])
			# Insertion sort changes only the presentation of unordered elements.
			value = values[i]
			j = i - 1
			while (j > 0 && values[j] > value) {
				values[j + 1] = values[j]
				j--
			}
			values[j + 1] = value
		}
		emit("\t\telements = {")
		for (i = 1; i <= n; i++)
			emit("\t\t\t" values[i] (i < n ? "," : ""))
		emit("\t\t}")
		next
	}
	emit($0)
	if ($0 == "\t}") {
		if (key in objects) exit 1
		objects[key] = block
		keys[++count] = key
	}
}
END {
	for (i = 2; i <= count; i++) {
		key = keys[i]
		j = i - 1
		while (j > 0 && keys[j] > key) {
			keys[j + 1] = keys[j]
			j--
		}
		keys[j + 1] = key
	}
	print header
	if (table_flags != "") print table_flags "\n"
	for (i = 1; i <= count; i++) {
		if (i > 1) print ""
		printf "%s", objects[keys[i]]
	}
	print "}"
}
