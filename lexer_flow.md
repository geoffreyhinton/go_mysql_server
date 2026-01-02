# Lexer State Machine Flow

This diagram shows the flow of the state machine lexer:

```
                                    ┌─────────────┐
                                    │    Start    │
                                    │  (lexLine)  │
                                    └─────┬───────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────┐
                    │              lexLine()                      │
                    │         (Main dispatcher)                   │
                    │   Reads next character and decides          │
                    │   which state to transition to              │
                    └─────────────────┬───────────────────────────┘
                                      │
                     ┌────────────────┼────────────────┐
                     │                │                │
                     ▼                ▼                ▼
        ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
        │   isSpace(r)    │ │   isEOL(r)      │ │  isLetter(r)    │
        │                 │ │                 │ │                 │
        │   lexSpaces     │ │    lexEOL       │ │ lexIdentifier   │
        │   ↓             │ │    ↓            │ │    ↓            │
        │ ignore spaces   │ │ handle newlines │ │ read identifier │
        │ return lexLine  │ │ return lexLine  │ │ emit ID/Keyword │
        └─────────────────┘ └─────────────────┘ │ return lexLine  │
                                                └─────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────────┐
│isAllowedInOp│ │unicode.Is   │ │ singleQuote(')  │
│    (r)      │ │ Digit(r)    │ │                 │
│             │ │             │ │ lexSingleQuote  │
│   lexOp     │ │  lexNumber  │ │       ↓         │
│     ↓       │ │      ↓      │ │  lexString()    │
│ read op     │ │ scan digits │ │ emit StringToken│
│ emit OpToken│ │ handle float│ │ return lexLine  │
│return lexLine│ │emit Int/Float│ └─────────────────┘
└─────────────┘ │return lexLine│
                └─────────────┘
        │
        ├─────────────────┬─────────────────┐
        ▼                 ▼                 ▼
┌─────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  quote(")   │ │ Simple tokens:  │ │   semiColon     │
│             │ │ comma, dot,     │ │                 │
│  lexQuote   │ │ leftParen,      │ │ emit EOFToken   │
│     ↓       │ │ rightParen      │ │  return nil     │
│lexString()  │ │                 │ │    (STOP)       │
│emit String  │ │ Emit token      │ └─────────────────┘
│return lexLine│ │ return lexLine  │
└─────────────┘ └─────────────────┘

                ┌─────────────────────┐
                │ Unexpected character│
                │                     │
                │    errorf()         │
                │ emit ErrorToken     │
                │   return nil        │
                │     (STOP)          │
                └─────────────────────┘
```

## State Functions Overview:

### Main States:
- **`lexLine`**: Entry point dispatcher that reads next character and routes to appropriate state
- **`lexSpaces`**: Consumes whitespace characters and ignores them
- **`lexEOL`**: Handles end-of-line characters and updates line numbers
- **`lexIdentifier`**: Reads letters/digits/underscores to form identifiers or keywords
- **`lexOp`**: Reads operator characters and validates against known operators
- **`lexNumber`**: Scans digits and handles integers and floating-point numbers
- **`lexQuote`/`lexSingleQuote`**: Entry points for string parsing
- **`lexString`**: Handles escape sequences and string termination

### Flow Control:
1. **Initialization**: `NewLexer()` sets initial state to `lexLine`
2. **Main Loop**: `Run()` calls current state function until it returns `nil`
3. **Token Access**: `Next()` provides sequential access to parsed tokens

### State Transitions:
- Most states return to `lexLine` after processing their token type
- Error states return `nil` to stop lexing
- EOF and semicolon also return `nil` to terminate

### Key Features:
- **Backtracking**: `backup()` allows states to "unread" characters
- **Token Emission**: `emit()` creates tokens with position information
- **Error Handling**: `errorf()` creates error tokens and stops lexing
- **Look-ahead**: `peekWord()` shows current accumulated characters