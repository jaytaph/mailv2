#!/bin/sh

REPO="github.com/bitmaelum/bitmaelum-server"

TOOLS="create-account hash-address jwt mail-server-config proof-of-work protect-account readmail sendmail"

# We use govvv to inject GIT version information into the applications
GO_PATH=`go env GOPATH`
GO_BUILD_FLAGS=`${GO_PATH}/bin/govvv build -pkg version -flags`

echo "Compiling [\c"

echo ".\c"
go build -ldflags "${GO_BUILD_FLAGS}" -o release/bitmaelum-server ${REPO}/bm-server
echo ".\c"
go build -ldflags "${GO_BUILD_FLAGS}" -o release/client ${REPO}/bm-client
echo ".\c"

for TOOL in $TOOLS; do
  go build -ldflags "${GO_BUILD_FLAGS}" -o release/${TOOL} ${REPO}/tools/${TOOL}
  echo ".\c"
done

echo "]"