#include "mvvm_inout.h"
#include <iostream>
#include <string>
#include <array>

#define ALPHABETSIZE 26
#define MULTIVACSIZE 8

byte alphabet[ALPHABETSIZE+1] = "abcdefghijklmnopqrstuvwxyz";
byte multivac[MULTIVACSIZE+1] = "MultiVAC";

using namespace std;

void test_mvvm_input_decode();
void test_mvvm_output_encode();

int main()
{
    test_mvvm_input_decode();
    test_mvvm_output_encode();
    return 0;
}

void test_mvvm_input_decode()
{
    /*
     * Data structure:
     *  mvvmInput{
     *      Params:    []byte("abcdefghijklmnopqrstuvwxyz"),
     *      Shard:     ShardIndex(1),
     *      Caller:    multivacaddress.Address{},
     *      MtvValue:  1,
     *      ShardData: []byte("abcdefghijklmnopqrstuvwxyz"),
     *      UserData:  []inputUserData{
     *          inputUserData{Data: []byte("abcdefghijklmnopqrstuvwxyz")},
     *          inputUserData{Data: []byte("0123456789")},
     *      },
     *  } 
     */

    byte bytes[] = {
        byte(248), byte(98), byte(154), byte(97), byte(98), byte(99), byte(100), byte(101), byte(102), byte(103),
        byte(104), byte(105), byte(106), byte(107), byte(108), byte(109), byte(110), byte(111), byte(112), byte(113),
        byte(114), byte(115), byte(116), byte(117), byte(118), byte(119), byte(120), byte(121), byte(122), byte(1),
        byte(128), byte(1), byte(154), byte(97), byte(98), byte(99), byte(100), byte(101), byte(102), byte(103),
        byte(104), byte(105), byte(106), byte(107), byte(108), byte(109), byte(110), byte(111), byte(112), byte(113),
        byte(114), byte(115), byte(116), byte(117), byte(118), byte(119), byte(120), byte(121), byte(122), byte(232),
        byte(219), byte(154), byte(97), byte(98), byte(99), byte(100), byte(101), byte(102), byte(103), byte(104),
        byte(105), byte(106), byte(107), byte(108), byte(109), byte(110), byte(111), byte(112), byte(113), byte(114),
        byte(115), byte(116), byte(117), byte(118), byte(119), byte(120), byte(121), byte(122), byte(203), byte(138),
        byte(48), byte(49), byte(50), byte(51), byte(52), byte(53), byte(54), byte(55), byte(56), byte(57)
    };
    dev::bytesConstRef bcr(bytes, sizeof(bytes) / sizeof(byte));
    
    // RLP decode using mvr::rlp_decode_to_mvvm_input.
    dev::RLP rlp(bcr);
    mvr::mvvm_input * input = mvr::rlp_decode_to_mvvm_input(bcr);

    // Print result.
    cout << "Check mvvmInput info:\n";
    cout << "-------- Params --------\n";
    auto params = input->get_parameters();
    for (auto p : params) { cout << p; } cout << endl;
    
    cout << "-------- Shard --------\n";
    cout << input->get_shard_index() << "\n";

    cout << "-------- Caller --------\n";
    auto caller = input->get_caller_address();
    cout << "[ "; for (auto c : caller) { cout << int(c) << " "; } cout << "]\n";

    cout << "-------- MtvValue --------\n";
    cout << input->get_mtv_value() << "\n";

    cout << "-------- ShardData --------\n";
    auto shard_data = input->get_input_shard_data();
    for (auto sd : shard_data) { cout << sd; } cout << endl;

    cout << "-------- UserData --------\n";
    auto user_data = input->get_input_user_data();
    auto size = user_data.size();
    for (int i = 0; i < size; i++) {
        cout << "User data #" << i << ": ";
        for (auto ud : user_data[i].data) { cout << ud; }
        cout << endl;
    }
    cout << endl;

    cout << "Done.\n\n";

    // delete the mvvm_input object.
    delete input;
}

void test_mvvm_output_encode()
{
    /*
     * Data structure:
     *  mvvmOutput{
     *      ShardData: []byte{
     *          42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42,
     *          42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42,
     *          42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42,
     *          42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42, 42,
     *      }, UserData: []outputUserData{
     *          outputUserData{
     *              Shard: 0, UserAddress: multivacaddress.Address{
     *                  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
     *              }, Data: []byte{
     *                  97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116,
     *                  117, 118, 119, 120, 121, 122,
     *              },
     *          }, outputUserData{
     *              Shard: 1, UserAddress: multivacaddress.Address{
     *                  255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
     *                  255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
     *              }, Data: []byte{
     *                  97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116,
     *                  117, 118, 119, 120, 121, 122,
     *              },
     *          },
     *      }, MtvData: []outputMtvData{
     *          outputMtvData{
     *              Shard: 0, UserAddress: multivacaddress.Address{
     *                  0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 255, 0, 255, 0, 255, 0, 255, 0,
     *                  255, 0, 255, 0, 255, 0, 255, 0,
     *              }, Value: 0,
     *          }, outputMtvData{
     *              Shard: 1, UserAddress: multivacaddress.Address{
     *                  255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 0, 255, 0, 255, 0, 255, 0, 255,
     *                  0, 255, 0, 255, 0, 255, 0, 255,
     *              }, Value: 65536,
     *          },
     *      }, ReduceData: []outputReduceData{
     *          outputReduceData{
     *              Shard:       1,
     *              Data: []byte{
     *                  97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116,
     *                  117, 118, 119, 120, 121, 122,
     *              },
     *              Api: []byte("MultiVAC"),
     *          },
     *      },
     *  }
     */
    byte addresses[][mvr::HASHSIZE] = {
    {
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    },
    {
        0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff,
        0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00,
    },
    {
        0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00,
        0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff,
    }
    };
    dev::bytes shard_data(100, 42);
    vector<mvr::output_user_data> user_data(2);
    user_data[0] = mvr::output_user_data{0, dev::bytes(mvr::HASHSIZE), dev::bytes(alphabet, alphabet+ALPHABETSIZE)};
    user_data[1] = mvr::output_user_data{
        1, dev::bytes(addresses[0], addresses[0]+mvr::HASHSIZE), dev::bytes(alphabet, alphabet+ALPHABETSIZE)
    };
    vector<mvr::output_mtv_data> mtv_data(2);
    mtv_data[0] = mvr::output_mtv_data{0, dev::bytes(addresses[1], addresses[1]+mvr::HASHSIZE), 0};
    mtv_data[1] = mvr::output_mtv_data{1, dev::bytes(addresses[2], addresses[2]+mvr::HASHSIZE), 65536};
    vector<mvr::output_reduce_data> reduce_data(1);
    reduce_data[0] = mvr::output_reduce_data{
        1, dev::bytes(), dev::bytes(alphabet, alphabet+ALPHABETSIZE), dev::bytes(multivac, multivac+MULTIVACSIZE)
    };
    mvr::mvvm_output output(shard_data, user_data, mtv_data, reduce_data);

    auto bytes = mvr::rlp_encode_from_mvvm_output(output);
    
    cout << "Check mvvmOutput code:\n";
    cout << "[ "; for (auto b : bytes) { cout << int(b) << ' '; } cout << "]\n";
    cout << "Done.\n\n";
}
