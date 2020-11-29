#ifndef _MVVM_INOUT_
#define _MVVM_INOUT_

#include <array>
#include <vector>

#include "RLP.h"

namespace mvr
{

/// Basic input output data structure for MVVM.

const size_t HASHSIZE = 32;
const int INPUT_ITEM_SIZE = 6;
const int OUTPUT_ITEM_SIZE = 3;

enum mvvm_input_fields  { FIELD_INPUT_PARAMS, FIELD_INPUT_SHARD_INDEX, FIELD_INPUT_ADDRESS, FIELD_INPUT_MTV_VALUE,
                          FIELD_INPUT_SHARD_DATA, FIELD_INPUT_USER_DATA };
enum mvvm_output_fields { FIELD_OUTPUT_SHARD_DATA, FIELD_OUTPUT_USER_DATA, FIELD_OUTPUT_MTV_DATA };

using shard_index = unsigned int;
typedef std::array<byte, HASHSIZE> hash;

/**
 * input_user_data records the necessary user data in the input data.
 */
typedef struct {
    dev::bytes data;
} input_user_data;

/**
 * output_user_data records the necessary user data in the output data.
 */
typedef struct {
    shard_index shard;
    dev::bytes useraddress;
    dev::bytes data;
} output_user_data;

/**
 * output_mtv_data records the necessary MTV coin cost in the output data.
 */
typedef struct {
    shard_index shard;
    dev::bytes useraddress;
    long value;
} output_mtv_data;

/**
 * output_reduce_data records the necessary data for reduce actions in the output data.
 */
typedef struct {
    shard_index shard;
    dev::bytes data;
    dev::bytes api;
} output_reduce_data;

/**
 * Class for transfering necessary shard data and user data from MVVM into life VM.
 */
class mvvm_input {
public:
    /// Constructe a null input.
    mvvm_input() {}

    /// Constructe an input given in the params, shard, etc.
    // TODO (wangruichao@mtv.ac): add other constructors if necessary.
    explicit mvvm_input(dev::bytes const p, shard_index index, dev::bytes const address, long value,
        dev::bytes const shard_data, std::vector<input_user_data> const user_data): params (p), shard (index), 
        caller (address), mtv_value (value), shard_data (shard_data), user_data (user_data) {}
    
    /// Destructor of the input.
    ~mvvm_input() {}
    
    /// The parameters in the function.
    dev::bytes get_parameters() const { return params; }

    /// The main index of the code.
    shard_index get_shard_index() const { return shard; }

    /// The caller of the function.
    dev::bytes const get_caller_address() const { return caller; }

    /// The value spent.
    long get_mtv_value() const { return mtv_value; }

    /// The related shard data.
    dev::bytes get_input_shard_data() const { return shard_data; }

    /// The related user data.
    std::vector<input_user_data> get_input_user_data() const { return user_data; }

private:
    dev::bytes params;                      // Parameter of the input.
    shard_index shard;                      // Shard index of the input.
    dev::bytes caller;                      // Caller address of the input.
    long mtv_value;                         // Total MTV value spent in executing the smart contract.
    dev::bytes shard_data;                  // Shard init data of the smart contract.
    std::vector<input_user_data> user_data; // User data of the smart contract.
};

/**
 * Class for transfering necessary shard data and user data from life VM into MVVM.
 */
class mvvm_output {
public:
    /// Constructe a null output.
    mvvm_output() {}

    /// Constructe an output given in the shard_data, user_data & mtv_data.
    // TODO (wangruichao@mtv.ac): add other constructors if necessary.
    explicit mvvm_output(dev::bytes const shard_data, std::vector<output_user_data> const user_data, 
        std::vector<output_mtv_data> const mtv_data, std::vector<output_reduce_data> const reduce_data):
        shard_data (shard_data), user_data (user_data), mtv_data (mtv_data), reduce_data (reduce_data) {}
    
    /// Destructor of the output.
    ~mvvm_output() {}
    
    /// The updated shard data.
    dev::bytes get_output_shard_data() const { return shard_data; }

    /// The added user data.
    std::vector<output_user_data> get_output_user_data() const { return user_data; }

    /// The cost MTV coin.
    std::vector<output_mtv_data> get_output_mtv_data() const { return mtv_data; }

    /// The reduce actions data.
    std::vector<output_reduce_data> get_output_reduce_data() const { return reduce_data; }

private:
    dev::bytes shard_data;                          // Shard data of the returned value.
    std::vector<output_user_data> user_data;        // User data of the returned value.
    std::vector<output_mtv_data> mtv_data;          // Change for 3rd-party cost of the returned value.
    std::vector<output_reduce_data> reduce_data;    // Data for reduce actions of the returned value.
};

mvvm_input * rlp_decode_to_mvvm_input(dev::bytesConstRef);
dev::bytes rlp_encode_from_mvvm_output(const mvvm_output &);
} // namespace mvr

#endif /* mvvm_inout.h */