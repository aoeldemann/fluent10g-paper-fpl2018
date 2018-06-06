ifeq ($(RTE_SDK),)
$(error "Please define RTE_SDK environment variable")
endif

RTE_TARGET ?= x86_64-native-linuxapp-gcc

include $(RTE_SDK)/mk/rte.vars.mk

APP = fluent10g_precision

SRCS-y :=
SRCS-$(CONFIG_RTE_LIBRTE_IEEE1588) += fluent10g_precision.c

CFLAGS += -O3 -I$(shell dirname `pwd`)/../lib
CFLAGS += $(WERROR_FLAGS)

include $(RTE_SDK)/mk/rte.extapp.mk
