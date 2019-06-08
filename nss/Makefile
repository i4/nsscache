# Compiler flags
LDLIBS  = -pthread
CFLAGS  = -Wall -Wextra -Wconversion
# DEB_BUILD_MAINT_OPTIONS='hardening=+all qa=+bug' dpkg-buildflags --export=make
CFLAGS   += -g -O2 -Werror=array-bounds -Werror=clobbered -Werror=volatile-register-var -Werror=implicit-function-declaration -fstack-protector-strong -Wformat -Werror=format-security
CPPFLAGS += -Wdate-time -D_FORTIFY_SOURCE=2
LDFLAGS  += -Wl,-z,relro -Wl,-z,now
# For tests
TEST_CFLAGS  += -fsanitize=address -fno-omit-frame-pointer -fsanitize=undefined
TEST_LDFLAGS += -fsanitize=address -fsanitize=undefined

all: libnss_cash.so.2

clean:
	rm -f libnss_cash.so.2 \
	    libcash_test.so tests/pw tests/passwd.nsscash

libnss_cash.so.2 libcash_test.so: $(wildcard *.c) $(wildcard *.h)
	$(CC) -o $@ -shared -fPIC -Wl,-soname,$@ \
		$(CFLAGS) $(CPPFLAGS) $(LDFLAGS) \
		file.c pw.c search.c \
		$(LDLIBS)


# Tests

test: tests/pw tests/passwd.nsscash
	LD_LIBRARY_PATH=. LD_PRELOD= ./tests/pw

tests/pw: tests/pw.c libcash_test.so
	$(CC) -o $@ $(CFLAGS) $(CPPFLAGS) $(LDFLAGS) \
		$(TEST_CFLAGS) $(TEST_LDFLAGS) -L. \
		$< $(LDLIBS) -lcash_test -lasan

tests/passwd.nsscash: tests/passwd
	../nsscash convert passwd $< $@

libcash_test.so: CFLAGS += $(TEST_CFLAGS)
libcash_test.so: CPPFLAGS += -DNSSCASH_PASSWD_FILE='"./tests/passwd.nsscash"'
libcash_test.so: LDFLAGS += $(TEST_LDFLAGS)

.PHONY: all clean test
