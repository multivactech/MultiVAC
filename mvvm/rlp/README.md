# C++ RLP Encoding/Decoding Library For MultiVAC

## 1. Introduce

The library is the encoding/decoding tool for MVVM (MultiVAC Virtual Machine).

The library implements the encoding method for the input data structure *mvvmInput* and for the output data structure *mvvmOutput*.

The library mainly depends on `libdevcore` library in project [aleth](https://github.com/ethereum/aleth).

## 2. Using the tool

The required C++ environment is listed here:

- C++11/C++17;
- STL;
- Boost;
- Aleth (included here).

## 3. Test

```C++
cd mvvm/rlp/
g++ -std=c++11 -o main.out test_rlp.cpp RLP.cpp mvvm_inout.cpp
./main.out
rm ./main.out
```
