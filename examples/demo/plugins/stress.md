---
nav:
  hidden: true
---
# Stress test for figures

This page holds big figures and many small ones, to check that the window
stays responsive while they draw. It is hidden from the sidebar. Scroll while
the figures draw: clicks and scrolling must keep working, and figures far
below the screen wait behind "Drawing diagram…" until you scroll towards them.
## Small figures above

~~~mermaid
graph LR
  A[Read] --> B[Edit] --> C[Share]
~~~

~~~lini
{ direction: row; gap: 30; }
|box#a| "One"
|box#b| "Two"
a -> b "next"
~~~

~~~mermaid
sequenceDiagram
  You->>App: open
  App-->>You: page
~~~

~~~lini
|mindmap| "Small" [
  |topic| "A"
  |topic| "B"
]
~~~

~~~mermaid
graph TD
  A{Saved?} -- yes --> B[Done]
  A -- no --> C[Save]
  C --> A
~~~

~~~lini
{ direction: row; gap: 30; }
|box#a| "One"
|box#b| "Two"
a -> b "next"
~~~

## A large flowchart

About 150 nodes in 6 subgraphs.

~~~mermaid
graph TD
  subgraph S0[Stage 1]
    direction TB
    N0_0[Step 1.1]
    N0_1[Step 1.2]
    N0_2[Step 1.3]
    N0_3[Step 1.4]
    N0_4[Step 1.5]
    N0_5[Step 1.6]
    N0_6[Step 1.7]
    N0_7[Step 1.8]
    N0_8[Step 1.9]
    N0_9[Step 1.10]
    N0_10[Step 1.11]
    N0_11[Step 1.12]
    N0_12[Step 1.13]
    N0_13[Step 1.14]
    N0_14[Step 1.15]
    N0_15[Step 1.16]
    N0_16[Step 1.17]
    N0_17[Step 1.18]
    N0_18[Step 1.19]
    N0_19[Step 1.20]
    N0_20[Step 1.21]
    N0_21[Step 1.22]
    N0_22[Step 1.23]
    N0_23[Step 1.24]
    N0_24[Step 1.25]
    N0_0 --> N0_1
    N0_1 --> N0_2
    N0_2 --> N0_3
    N0_3 --> N0_4
    N0_4 --> N0_5
    N0_5 --> N0_6
    N0_6 --> N0_7
    N0_7 --> N0_8
    N0_8 --> N0_9
    N0_9 --> N0_10
    N0_10 --> N0_11
    N0_11 --> N0_12
    N0_12 --> N0_13
    N0_13 --> N0_14
    N0_14 --> N0_15
    N0_15 --> N0_16
    N0_16 --> N0_17
    N0_17 --> N0_18
    N0_18 --> N0_19
    N0_19 --> N0_20
    N0_20 --> N0_21
    N0_21 --> N0_22
    N0_22 --> N0_23
    N0_23 --> N0_24
  end
  subgraph S1[Stage 2]
    direction TB
    N1_0[Step 2.1]
    N1_1[Step 2.2]
    N1_2[Step 2.3]
    N1_3[Step 2.4]
    N1_4[Step 2.5]
    N1_5[Step 2.6]
    N1_6[Step 2.7]
    N1_7[Step 2.8]
    N1_8[Step 2.9]
    N1_9[Step 2.10]
    N1_10[Step 2.11]
    N1_11[Step 2.12]
    N1_12[Step 2.13]
    N1_13[Step 2.14]
    N1_14[Step 2.15]
    N1_15[Step 2.16]
    N1_16[Step 2.17]
    N1_17[Step 2.18]
    N1_18[Step 2.19]
    N1_19[Step 2.20]
    N1_20[Step 2.21]
    N1_21[Step 2.22]
    N1_22[Step 2.23]
    N1_23[Step 2.24]
    N1_24[Step 2.25]
    N1_0 --> N1_1
    N1_1 --> N1_2
    N1_2 --> N1_3
    N1_3 --> N1_4
    N1_4 --> N1_5
    N1_5 --> N1_6
    N1_6 --> N1_7
    N1_7 --> N1_8
    N1_8 --> N1_9
    N1_9 --> N1_10
    N1_10 --> N1_11
    N1_11 --> N1_12
    N1_12 --> N1_13
    N1_13 --> N1_14
    N1_14 --> N1_15
    N1_15 --> N1_16
    N1_16 --> N1_17
    N1_17 --> N1_18
    N1_18 --> N1_19
    N1_19 --> N1_20
    N1_20 --> N1_21
    N1_21 --> N1_22
    N1_22 --> N1_23
    N1_23 --> N1_24
  end
  subgraph S2[Stage 3]
    direction TB
    N2_0[Step 3.1]
    N2_1[Step 3.2]
    N2_2[Step 3.3]
    N2_3[Step 3.4]
    N2_4[Step 3.5]
    N2_5[Step 3.6]
    N2_6[Step 3.7]
    N2_7[Step 3.8]
    N2_8[Step 3.9]
    N2_9[Step 3.10]
    N2_10[Step 3.11]
    N2_11[Step 3.12]
    N2_12[Step 3.13]
    N2_13[Step 3.14]
    N2_14[Step 3.15]
    N2_15[Step 3.16]
    N2_16[Step 3.17]
    N2_17[Step 3.18]
    N2_18[Step 3.19]
    N2_19[Step 3.20]
    N2_20[Step 3.21]
    N2_21[Step 3.22]
    N2_22[Step 3.23]
    N2_23[Step 3.24]
    N2_24[Step 3.25]
    N2_0 --> N2_1
    N2_1 --> N2_2
    N2_2 --> N2_3
    N2_3 --> N2_4
    N2_4 --> N2_5
    N2_5 --> N2_6
    N2_6 --> N2_7
    N2_7 --> N2_8
    N2_8 --> N2_9
    N2_9 --> N2_10
    N2_10 --> N2_11
    N2_11 --> N2_12
    N2_12 --> N2_13
    N2_13 --> N2_14
    N2_14 --> N2_15
    N2_15 --> N2_16
    N2_16 --> N2_17
    N2_17 --> N2_18
    N2_18 --> N2_19
    N2_19 --> N2_20
    N2_20 --> N2_21
    N2_21 --> N2_22
    N2_22 --> N2_23
    N2_23 --> N2_24
  end
  subgraph S3[Stage 4]
    direction TB
    N3_0[Step 4.1]
    N3_1[Step 4.2]
    N3_2[Step 4.3]
    N3_3[Step 4.4]
    N3_4[Step 4.5]
    N3_5[Step 4.6]
    N3_6[Step 4.7]
    N3_7[Step 4.8]
    N3_8[Step 4.9]
    N3_9[Step 4.10]
    N3_10[Step 4.11]
    N3_11[Step 4.12]
    N3_12[Step 4.13]
    N3_13[Step 4.14]
    N3_14[Step 4.15]
    N3_15[Step 4.16]
    N3_16[Step 4.17]
    N3_17[Step 4.18]
    N3_18[Step 4.19]
    N3_19[Step 4.20]
    N3_20[Step 4.21]
    N3_21[Step 4.22]
    N3_22[Step 4.23]
    N3_23[Step 4.24]
    N3_24[Step 4.25]
    N3_0 --> N3_1
    N3_1 --> N3_2
    N3_2 --> N3_3
    N3_3 --> N3_4
    N3_4 --> N3_5
    N3_5 --> N3_6
    N3_6 --> N3_7
    N3_7 --> N3_8
    N3_8 --> N3_9
    N3_9 --> N3_10
    N3_10 --> N3_11
    N3_11 --> N3_12
    N3_12 --> N3_13
    N3_13 --> N3_14
    N3_14 --> N3_15
    N3_15 --> N3_16
    N3_16 --> N3_17
    N3_17 --> N3_18
    N3_18 --> N3_19
    N3_19 --> N3_20
    N3_20 --> N3_21
    N3_21 --> N3_22
    N3_22 --> N3_23
    N3_23 --> N3_24
  end
  subgraph S4[Stage 5]
    direction TB
    N4_0[Step 5.1]
    N4_1[Step 5.2]
    N4_2[Step 5.3]
    N4_3[Step 5.4]
    N4_4[Step 5.5]
    N4_5[Step 5.6]
    N4_6[Step 5.7]
    N4_7[Step 5.8]
    N4_8[Step 5.9]
    N4_9[Step 5.10]
    N4_10[Step 5.11]
    N4_11[Step 5.12]
    N4_12[Step 5.13]
    N4_13[Step 5.14]
    N4_14[Step 5.15]
    N4_15[Step 5.16]
    N4_16[Step 5.17]
    N4_17[Step 5.18]
    N4_18[Step 5.19]
    N4_19[Step 5.20]
    N4_20[Step 5.21]
    N4_21[Step 5.22]
    N4_22[Step 5.23]
    N4_23[Step 5.24]
    N4_24[Step 5.25]
    N4_0 --> N4_1
    N4_1 --> N4_2
    N4_2 --> N4_3
    N4_3 --> N4_4
    N4_4 --> N4_5
    N4_5 --> N4_6
    N4_6 --> N4_7
    N4_7 --> N4_8
    N4_8 --> N4_9
    N4_9 --> N4_10
    N4_10 --> N4_11
    N4_11 --> N4_12
    N4_12 --> N4_13
    N4_13 --> N4_14
    N4_14 --> N4_15
    N4_15 --> N4_16
    N4_16 --> N4_17
    N4_17 --> N4_18
    N4_18 --> N4_19
    N4_19 --> N4_20
    N4_20 --> N4_21
    N4_21 --> N4_22
    N4_22 --> N4_23
    N4_23 --> N4_24
  end
  subgraph S5[Stage 6]
    direction TB
    N5_0[Step 6.1]
    N5_1[Step 6.2]
    N5_2[Step 6.3]
    N5_3[Step 6.4]
    N5_4[Step 6.5]
    N5_5[Step 6.6]
    N5_6[Step 6.7]
    N5_7[Step 6.8]
    N5_8[Step 6.9]
    N5_9[Step 6.10]
    N5_10[Step 6.11]
    N5_11[Step 6.12]
    N5_12[Step 6.13]
    N5_13[Step 6.14]
    N5_14[Step 6.15]
    N5_15[Step 6.16]
    N5_16[Step 6.17]
    N5_17[Step 6.18]
    N5_18[Step 6.19]
    N5_19[Step 6.20]
    N5_20[Step 6.21]
    N5_21[Step 6.22]
    N5_22[Step 6.23]
    N5_23[Step 6.24]
    N5_24[Step 6.25]
    N5_0 --> N5_1
    N5_1 --> N5_2
    N5_2 --> N5_3
    N5_3 --> N5_4
    N5_4 --> N5_5
    N5_5 --> N5_6
    N5_6 --> N5_7
    N5_7 --> N5_8
    N5_8 --> N5_9
    N5_9 --> N5_10
    N5_10 --> N5_11
    N5_11 --> N5_12
    N5_12 --> N5_13
    N5_13 --> N5_14
    N5_14 --> N5_15
    N5_15 --> N5_16
    N5_16 --> N5_17
    N5_17 --> N5_18
    N5_18 --> N5_19
    N5_19 --> N5_20
    N5_20 --> N5_21
    N5_21 --> N5_22
    N5_22 --> N5_23
    N5_23 --> N5_24
  end
  N0_24 --> N1_0
  N1_24 --> N2_0
  N2_24 --> N3_0
  N3_24 --> N4_0
  N4_24 --> N5_0
~~~

## Small figures between

~~~mermaid
graph LR
  A[Read] --> B[Edit] --> C[Share]
~~~

~~~lini
{ direction: row; gap: 30; }
|box#a| "One"
|box#b| "Two"
a -> b "next"
~~~

~~~mermaid
sequenceDiagram
  You->>App: open
  App-->>You: page
~~~

~~~lini
|mindmap| "Small" [
  |topic| "A"
  |topic| "B"
]
~~~

## A large schematic

About 60 parts.

~~~lini
{
  layout: schematic;
  |v5::label| { symbol: power; } [ "5V" ]
}

// A chain of 12 stages: 60 parts in all.
|component#U1| "STAGE1" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R1| "1k"
|C#C1| "100n"
|LED#D1| "S1"
|R#Q1| "330"
|component#U2| "STAGE2" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R2| "2k"
|C#C2| "100n"
|LED#D2| "S2"
|R#Q2| "330"
|component#U3| "STAGE3" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R3| "3k"
|C#C3| "100n"
|LED#D3| "S3"
|R#Q3| "330"
|component#U4| "STAGE4" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R4| "4k"
|C#C4| "100n"
|LED#D4| "S4"
|R#Q4| "330"
|component#U5| "STAGE5" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R5| "5k"
|C#C5| "100n"
|LED#D5| "S5"
|R#Q5| "330"
|component#U6| "STAGE6" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R6| "6k"
|C#C6| "100n"
|LED#D6| "S6"
|R#Q6| "330"
|component#U7| "STAGE7" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R7| "7k"
|C#C7| "100n"
|LED#D7| "S7"
|R#Q7| "330"
|component#U8| "STAGE8" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R8| "8k"
|C#C8| "100n"
|LED#D8| "S8"
|R#Q8| "330"
|component#U9| "STAGE9" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R9| "9k"
|C#C9| "100n"
|LED#D9| "S9"
|R#Q9| "330"
|component#U10| "STAGE10" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R10| "10k"
|C#C10| "100n"
|LED#D10| "S10"
|R#Q10| "330"
|component#U11| "STAGE11" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R11| "11k"
|C#C11| "100n"
|LED#D11| "S11"
|R#Q11| "330"
|component#U12| "STAGE12" [
  |pin#a| { side: left; number: 1; }
  |pin#b| { side: right; number: 2; }
  |pin#g| { side: bottom; number: 3; }
]
|R#R12| "12k"
|C#C12| "100n"
|LED#D12| "S12"
|R#Q12| "330"

U1.a - R1 - |v5|
U1.g - |gnd|
U1.b - C1 - |gnd|
U1.b - Q1 - D1 - |gnd|
U1.b - U2.a
U2.a - R2 - |v5|
U2.g - |gnd|
U2.b - C2 - |gnd|
U2.b - Q2 - D2 - |gnd|
U2.b - U3.a
U3.a - R3 - |v5|
U3.g - |gnd|
U3.b - C3 - |gnd|
U3.b - Q3 - D3 - |gnd|
U3.b - U4.a
U4.a - R4 - |v5|
U4.g - |gnd|
U4.b - C4 - |gnd|
U4.b - Q4 - D4 - |gnd|
U4.b - U5.a
U5.a - R5 - |v5|
U5.g - |gnd|
U5.b - C5 - |gnd|
U5.b - Q5 - D5 - |gnd|
U5.b - U6.a
U6.a - R6 - |v5|
U6.g - |gnd|
U6.b - C6 - |gnd|
U6.b - Q6 - D6 - |gnd|
U6.b - U7.a
U7.a - R7 - |v5|
U7.g - |gnd|
U7.b - C7 - |gnd|
U7.b - Q7 - D7 - |gnd|
U7.b - U8.a
U8.a - R8 - |v5|
U8.g - |gnd|
U8.b - C8 - |gnd|
U8.b - Q8 - D8 - |gnd|
U8.b - U9.a
U9.a - R9 - |v5|
U9.g - |gnd|
U9.b - C9 - |gnd|
U9.b - Q9 - D9 - |gnd|
U9.b - U10.a
U10.a - R10 - |v5|
U10.g - |gnd|
U10.b - C10 - |gnd|
U10.b - Q10 - D10 - |gnd|
U10.b - U11.a
U11.a - R11 - |v5|
U11.g - |gnd|
U11.b - C11 - |gnd|
U11.b - Q11 - D11 - |gnd|
U11.b - U12.a
U12.a - R12 - |v5|
U12.g - |gnd|
U12.b - C12 - |gnd|
U12.b - Q12 - D12 - |gnd|
~~~

## Small figures below

~~~mermaid
graph LR
  A[Read] --> B[Edit] --> C[Share]
~~~

~~~lini
{ direction: row; gap: 30; }
|box#a| "One"
|box#b| "Two"
a -> b "next"
~~~

~~~mermaid
sequenceDiagram
  You->>App: open
  App-->>You: page
~~~

~~~lini
|mindmap| "Small" [
  |topic| "A"
  |topic| "B"
]
~~~

~~~mermaid
graph TD
  A{Saved?} -- yes --> B[Done]
  A -- no --> C[Save]
  C --> A
~~~

~~~lini
{ direction: row; gap: 30; }
|box#a| "One"
|box#b| "Two"
a -> b "next"
~~~

~~~mermaid
graph LR
  A[Read] --> B[Edit] --> C[Share]
~~~

~~~lini
|mindmap| "Small" [
  |topic| "A"
  |topic| "B"
]
~~~

