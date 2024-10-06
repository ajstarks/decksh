# dshfmt - a formatter and checker for decksh files

dshfmt formats decksh files into a consistent format, placing the output on standard output
or writing over the original file. Input is read from a named file, 
or standard input if no file is specified (no re-writing in this case).

deck..edeck, slide..eslide, for..efor, if..eif, def..edef and list structures are indented,
and integrity checks (matching pairs, closing lists) are performed. No formatting is done
is any inconsistencies are found.

The exit status reflects the number of issues found.


## options

```
dshfmt [options] file
 -dump        dump raw parsed data
 -fsort       show keyword frequencies
 -ksort       show keyword counts
 -i string    indent string (default "\t")
 -v           (dump, fsort and ksort options combined)
 -w           rewrite the source
 ```

 Given the file
 ```
 deck
 slide
 x=10
 x+=20
 dchart -xlabel=10 -yaxis "foo.d"
 ctext "hello, world" 20 20 5 "sans" "red" 100 // comment
 for y=10 50 5
 ctext "hi" x y
 efor
 list 5 20
 li "hello"
  li "world"
 elist
 blist 10 20
 li "foo"
 li "bar"
 elist
 blist 50 20
 li "extra blist"
 elist
 clist 20 20
 li "foo"
 elist
 nlist 30 20
 li "foo"
 li "bar"
 elist
 eslide
 edeck
 ```

 dshfmt produces:

 ```
 deck

	slide
		x      = 10
		x      += 20
		dchart -xlabel = 10 -yaxis "foo.d"
		ctext  "hello, world" 20 20 5 "sans" "red" 100 // comment
		for    y = 10 50 5
			ctext  "hi"           x y
		efor

		list   5 20
			li    "hello"
			li    "world"
		elist

		blist  10 20
			li    "foo"
			li    "bar"
		elist

		blist  50 20
			li    "extra blist"
		elist

		clist  20 20
			li    "foo"
		elist

		nlist  30 20
			li    "foo"
			li    "bar"
		elist

	eslide

 edeck
 ```


 The -dump option shows the parsed tokens and line numbers:
 ```
 Line Len Tokens
    1   1 [deck]
    2   1 [slide]
    3   3 [x = 10]
    4   4 [x + = 20]
    5   8 [dchart - xlabel = 10 - yaxis "foo.d"]
    6   9 [ctext "hello, world" 20 20 5 "sans" "red" 100 // comment]
    7   6 [for y = 10 50 5]
    8   4 [ctext "hi" x y]
    9   1 [efor]
   10   3 [list 5 20]
   11   2 [li "hello"]
   12   2 [li "world"]
   13   1 [elist]
   14   3 [blist 10 20]
   15   2 [li "foo"]
   16   2 [li "bar"]
   17   1 [elist]
   18   3 [blist 50 20]
   19   2 [li "extra blist"]
   20   1 [elist]
   21   3 [clist 20 20]
   22   2 [li "foo"]
   23   1 [elist]
   24   3 [nlist 30 20]
   25   2 [li "foo"]
   26   2 [li "bar"]
   27   1 [elist]
   28   1 [eslide]
   29   1 [edeck]
   ```

 The -fsort option lists the keywords by line frequency.
 ```
 Keyword    Freq Lines
 li            8 [11 12 15 16 19 22 25 26]
 elist         5 [13 17 20 23 27]
 ctext         2 [6 8]
 blist         2 [14 18]
 nlist         1 [24]
 edeck         1 [29]
 eslide        1 [28]
 dchart        1 [5]
 list          1 [10]
 clist         1 [21]
 efor          1 [9]
 deck          1 [1]
 slide         1 [2]
 for           1 [7]
 ```

 The -ksort option lists the occurance sorted alphabetically

 ```
 Keyword    Freq Lines
 blist         2 [14 18]
 clist         1 [21]
 ctext         2 [6 8]
 dchart        1 [5]
 deck          1 [1]
 edeck         1 [29]
 efor          1 [9]
 elist         5 [13 17 20 23 27]
 eslide        1 [28]
 for           1 [7]
 li            8 [11 12 15 16 19 22 25 26]
 list          1 [10]
 nlist         1 [24]
 slide         1 [2]
 ```
