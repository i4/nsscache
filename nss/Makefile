# Compiler flags
LDLIBS  = -pthread
CFLAGS  = -fPIC -Wall -Wextra -Wconversion
LDFLAGS = -shared
# DEB_BUILD_MAINT_OPTIONS='hardening=+all qa=+bug' dpkg-buildflags --export=make
CFLAGS   += -g -O2 -Werror=array-bounds -Werror=clobbered -Werror=volatile-register-var -Werror=implicit-function-declaration -fstack-protector-strong -Wformat -Werror=format-security
CPPFLAGS += -Wdate-time -D_FORTIFY_SOURCE=2
LDFLAGS  += -Wl,-z,relro -Wl,-z,now

# During development
#CFLAGS  += -fsanitize=address -fno-omit-frame-pointer -fsanitize=undefined
#LDFLAGS += -fsanitize=address -fsanitize=undefined

all: libnss_cash.so.2

clean:
	rm -f libnss_cash.so.2

libnss_cash.so.2: $(wildcard *.c) $(wildcard *.h)
	$(CC) -o $@ -Wl,-soname,$@ $(CFLAGS) $(CPPFLAGS) $(LDFLAGS) \
		file.c pw.c search.c \
		$(LDLIBS)

.PHONY: all clean
