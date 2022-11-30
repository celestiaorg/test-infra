.ONESHELL: #

SHELL=/bin/bash
.SHELLFLAGS += -e
PROJECTNAME=$(shell basename "$(PWD)")
DIR_FULLPATH=$(shell pwd)
RUNNER ?=local-docker
# RUNNER can also be cluster-k8s
TG ?=$(HOME)/go/bin/testground

## help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

## check-git: Check if git is installed on the machine
check-git:
ifeq (,$(shell which git))	
	@echo "git is not installed, you must install git first."
	exit 1;
endif
.PHONY: check-git

## check-go: Check if go is installed on the machine
check-go:
ifeq (,$(shell which go))	
	@echo "go is not installed, you must install go first."
	exit 1;
endif
.PHONY: check-go

## check-gcc: Check if gcc is installed on the machine
check-gcc:
ifeq (,$(shell which gcc))	
	@echo "gcc is not installed, you must install go first."
	exit 1;
endif
.PHONY: check-gcc

## check-docker: Check if docker is installed on the machine
check-docker:
ifeq (,$(shell which docker))	
	@echo "docker is not installed, you must install go first."
	exit 1;
else
ifeq (/snap/bin/docker,$(shell which docker))
	@echo "You have docker installed through snap. This is problematic, snap won't let `install-tg` do its job properly with Docker. Aborting"
	exit 1;
endif
endif
.PHONY: check-docker

check: check-git check-go check-gcc check-docker
	@echo > /dev/null
.PHONY: check

$(TG): check
	@echo "Do you want to install to ${HOME}/go/bin/testground? (y/n):"
	@read line; if [ $$line != "y" ]; then echo "Abandoned testground install"; exit 1 ; fi
	@git clone --depth 1 https://github.com/testground/testground.git /tmp/testground
	@cd /tmp/testground
	@make install
	@rm -rf /tmp/testground
	@echo "Done."

## check-composition-arg: Check if COMPOSITION env var was provided
check-testplan-arg:
ifeq (,${TESTPLAN})
	@printf "You must specify a testplan, example:\n\t make COMMAND TESTPLAN=001-big-blocks\n\n"
	exit 1
endif
.PHONY: check-testplan-arg

## check-composition-arg: Check if COMPOSITION env var was provided
check-runner-arg:
ifeq (,${RUNNER})
	@printf "You must specify which runner you want to use, example:\n\t make COMMAND RUNNER=local-docker \n\n"
	exit 1
endif
.PHONY: check-runner-arg

## check-composition-arg: check if composition env var was provided
check-composition-arg:
ifeq (,${COMPOSITION})
	@printf "you must specify a testplan, example:\n\t make COMMAND COMPOSITION=pdf-8\n\n"
	exit 1
endif
.phony: check-composition-arg

## check-composition-arg: check if composition env var was provided
check-name-arg:
ifeq (,${NAME})
	@printf "you must specify a testplan, example:\n\t make COMMAND NAME=celestia\n\n"
	exit 1
endif
.phony: check-composition-arg

## tg-start: Start the deamon
tg-start:
	$(TGPATH) daemon
.PHONY: tg-start

## tg-create-testplan: Create test plan under ./plans/ of this repository
tg-create-testplan: $(TG) check-name-arg
	TESTGROUND_HOME=${DIR_FULLPATH} $(TG) plan create --plan=${NAME}
	@rm -rf ./data ./sdks
	@mkdir ./docs/test-plans/${NAME}
	@cp ./docs/test-plans/tp-template.md ./docs/test-plans/${NAME}/${NAME}.md
.PHONY: tg-create-testplan

## tg-import-testplan: Import testplan to TESTGROUND_HOME
tg-import-testplan: $(TG) check-testplan-arg check-name-arg
	$(TG) plan import --from ./plans/${TESTPLAN} --name ${NAME}
.PHONY: tg-import-testplan

## tg-run-composition: runs a specific composition by name given a testplan and a runner
tg-run-composition: $(TG) check-testplan-arg check-runner-arg check-composition-arg
	$(TG) run composition \
		-f plans/${TESTPLAN}/compositions/${RUNNER}/${COMPOSITION}.toml \
		--wait
.PHONY: tg-run-composition

## tg-run: alias to tg-run-composition
tg-run: tg-run-composition
.PHONY: tg-run
