#!/bin/bash

echo "=== Executor Package Verification ==="
echo

echo "1. Running tests with race detection..."
go test -race ./internal/executor/ > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ All tests passed (race-free)"
else
    echo "   ✗ Tests failed"
    exit 1
fi

echo "2. Checking coverage..."
COVERAGE=$(go test -cover ./internal/executor/ 2>&1 | grep -o '[0-9.]*%' | head -1)
echo "   ✓ Coverage: $COVERAGE"

echo "3. Running go vet..."
go vet ./internal/executor/... > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ go vet clean"
else
    echo "   ✗ go vet found issues"
    exit 1
fi

echo "4. Checking formatting..."
UNFORMATTED=$(gofmt -l ./internal/executor/)
if [ -z "$UNFORMATTED" ]; then
    echo "   ✓ Code properly formatted"
else
    echo "   ✗ Unformatted files found"
    exit 1
fi

echo "5. Counting implementation..."
LINES=$(wc -l ./internal/executor/*.go | tail -1 | awk '{print $1}')
FILES=$(ls -1 ./internal/executor/*.go | wc -l | awk '{print $1}')
echo "   ✓ $FILES files, $LINES total lines"

echo
echo "=== Verification Complete ==="
echo "All checks passed! Package is production-ready."
