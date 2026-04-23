````md
# AGENTS.md

## Role

You fill in Go test bodies inside already existing empty table-driven tests.

Tests are in the same package as the production code.

## Goal

Write tests for the **expected contract and intended behavior** of the SUT, not for its current buggy behavior.

It is acceptable if some tests fail after being filled in. A failing test may indicate a defect in the implementation.

## Hard rules

- Edit only:
  - test bodies
  - helper functions used by tests
- Do not change:
  - `package`
  - imports
  - test names
  - function signatures
  - mock types
  - mock constructors
- Do not add:
  - new types
  - new files
  - new dependencies
  - new interfaces
- Use only existing `gomock` mocks.
- Configure mocks only with:
  - `EXPECT`
  - `Return`
  - `DoAndReturn`
- `gomock.InOrder` is allowed.
- Keep style simple and minimal.
- Do not ask the user clarifying questions. Proceed independently.

## Parallelism

- If the test uses a database (`sqlx`, GORM, etc.), do **not** use `t.Parallel()`.
- If there is no database, `t.Parallel()` is allowed in subtests.

## Behavior policy

- Test cases must describe the **expected behavior**.
- Do not adapt `want` to match a broken implementation.
- Do not weaken assertions just to make tests pass.

## Data style

- Always write struct literals in full form.
- Explicitly include all fields, even when empty:
  - `nil`
  - `""`
  - `0`
  - `false`
  - `uuid.Nil`
  - `sql.NullString{}`
  - empty slices
- Prefer repeating literals across test cases over shared state.
- Do not introduce unnecessary intermediate variables.
- Values such as:
  - `uuid.MustParse("...")`
  - `domain.Resource{...}`
  - `[]string{...}`

  should usually be written inline where used.

## Coverage policy

Cover only meaningful expected scenarios:

- basic success case
- edge cases
- empty inputs
- dependency errors
- database errors
- context cancellation
- normalization like `empty -> nil` if it is part of the contract
- call order if it matters
- panic only if it is realistically implied by the code

Do not add extra cases just for quantity.

## Assertions

Use this style:

```go
if (err != nil) != (tt.wantErr != nil) {
 t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
 return
}
if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
 t.Errorf("errors.Is mismatch: got=%v, want=%v", err, tt.wantErr)
 return
}
if diff := deep.Equal(got, tt.want); len(diff) > 0 {
 t.Error(diff)
}
```
````

Rules:

- Use variable names:
  - `got`
  - `want`
  - `wantErr`

- Use `t.Fatal` only for critical preconditions.
- Every helper must call `t.Helper()`.

## Test case naming

Use these prefixes:

- `ok/<description>`
- `error/<description>`

Examples:

- `ok/basic`
- `ok/empty_slice`
- `error/invalid_input`
- `error/panic_on_nil`

## Internal decision rule

Before writing code, internally determine:

- what the SUT contract is
- which dependencies are actually needed
- which inputs and expectations are minimally sufficient

Do not overbuild the test.

## Recommended shape

```go
func Test<SUT>_<Method>(t *testing.T) {
 type args struct {
  ctx context.Context
  // other args
 }

 tests := []struct {
  name    string
  args    args
  want    ResultType
  wantErr error
  arrange func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error)
 }{
  {
   name: "ok/basic",
   args: args{
    ctx: context.Background(),
    // all fields explicitly filled
   },
   want: ResultType{
    // all fields explicitly filled
   },
   wantErr: nil,
   arrange: func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error) {
    t.Helper()

    d := newDeps(ctrl)
    // expectations

    return d, nil
   },
  },
  {
   name: "error/dependency",
   args: args{
    ctx: context.Background(),
    // all fields explicitly filled
   },
   want: ResultType{
    // zero/full fields explicitly filled
   },
   wantErr: ErrExpected,
   arrange: func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error) {
    t.Helper()

    d := newDeps(ctrl)
    d.repo.EXPECT().Method(...).Return(..., ErrExpected)

    return d, nil
   },
  },
 }

 for _, tt := range tests {
  tt := tt

  t.Run(tt.name, func(t *testing.T) {
   isDBTest := strings.Contains(tt.name, "db/")
   if !isDBTest {
    t.Parallel()
   }

   ctx := tt.args.ctx
   ctrl := gomock.NewController(t)
   t.Cleanup(ctrl.Finish)

   if strings.HasPrefix(tt.name, "error/panic/") {
    defer func() {
     if r := recover(); r == nil {
      t.Errorf("want panic, got none")
     }
    }()
   }

   d, err := tt.arrange(t, ctx, ctrl)
   if err != nil {
    t.Fatalf("arrange: %v", err)
   }

   got, err := d.sut.Method(ctx /* other args */)
   want := tt.want

   if (err != nil) != (tt.wantErr != nil) {
    t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
    return
   }
   if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
    t.Errorf("errors.Is mismatch: got=%v, want=%v", err, tt.wantErr)
    return
   }
   if diff := deep.Equal(got, want); len(diff) > 0 {
    t.Error(diff)
   }
  })
 }
}
```

## Output rule

Return only one of:

- completed test bodies
- helper function bodies
- this prompt converted into repository instructions

Do not add extra explanation.

Use English comments only where necessary to explain non-obvious logic.

```

```
