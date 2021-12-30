hi Comment ctermfg=darkgray
hi Statement ctermfg=darkred
hi String ctermfg=darkblue
syn keyword    Statement    deck edeck canvas include def edef grid call call func func vmap area slide eslide text ctext etext rtext random textfile textblock textbox  textcode lbrace rbrace ubrace dbrace lbracket rbracket ubracket dbracket blist list nlist clist li elist data edata dchart for efor legend image cimage polygon rect roundrect rrect square ellipse acircle circle line curve arc arrow lcarrow dcarrow rcarrow ucarrow hline vline polarx polary polar sprint format star pill arctext sqrt
syn region    Comment    start="//" end="$"

" Go escapes
syn match       goEscapeOctal       display contained "\\[0-7]\{3}"
syn match       goEscapeC           display contained +\\[abfnrtv\\'"]+
syn match       goEscapeX           display contained "\\x\x\{2}"
syn match       goEscapeU           display contained "\\u\x\{4}"
syn match       goEscapeBigU        display contained "\\U\x\{8}"
syn match       goEscapeError       display contained +\\[^0-7xuUabfnrtv\\'"]+

hi def link     goEscapeOctal       goSpecialString
hi def link     goEscapeC           goSpecialString
hi def link     goEscapeX           goSpecialString
hi def link     goEscapeU           goSpecialString
hi def link     goEscapeBigU        goSpecialString
hi def link     goSpecialString     Special
hi def link     goEscapeError       Error

" Strings and their contents
syn cluster     goStringGroup       contains=goEscapeOctal,goEscapeC,goEscapeX,goEscapeU,goEscapeBigU,goEscapeError
syn region      goString            start=+"+ skip=+\\\\\|\\"+ end=+"+ contains=@goStringGroup
syn region      goRawString         start=+`+ end=+`+
hi def link     goString            String
hi def link     goRawString         String
