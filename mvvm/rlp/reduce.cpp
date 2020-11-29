// origin code (C++):
// compile: emcc -std=c++11 -g -s WASM=1 -o reduce.html reduce.cpp mvvm_inout.cpp RLP.cpp -v -I /usr/local/Cellar/boost/1.69.0_2/include/ -s EXPORTED_FUNCTIONS="['__Z4execPhi','__Z11reduce_execPhi']"
#include <stdio.h>
#include <vector>

#include "mvvm_inout.h"

#define MONEY_TO_SPEND 1000

using namespace std;

extern dev::bytes * exec(byte * bytes, int size)
{
    dev::bytesConstRef bcr(bytes, size);

    /// Start function here
    auto input = mvr::rlp_decode_to_mvvm_input(bcr);
    auto shard = input->get_shard_index();
    auto caller = input->get_caller_address();
    auto val = input->get_mtv_value();
    auto shard_data = input->get_input_shard_data();
    auto user = input->get_input_user_data();
    printf("value: %ld, shard data: ", val);
    for (auto s : shard_data) { printf("%c ", s); } printf("\n");

    dev::bytes shard_out(shard_data.begin(), shard_data.end() - 2);
    vector<mvr::output_user_data> out_user;
    vector<mvr::output_mtv_data> mtv;
    vector<mvr::output_reduce_data> reduce_action;

    dev::bytes od(user[0].data.begin(), user[0].data.end() - 2);
    out_user.push_back(mvr::output_user_data{shard, caller, od});
    if (val < MONEY_TO_SPEND) { /* Here is an exception... */ }
    else mtv.push_back(mvr::output_mtv_data{shard, caller, MONEY_TO_SPEND});

    byte funcname[] = {byte(95),byte(95),byte(90),byte(49),byte(49),byte(114),byte(101),byte(100),byte(117),byte(99),byte(101),byte(95),byte(101),byte(120),byte(101),byte(99),byte(80),byte(104),byte(105)};
    dev::bytes reduce_data(user[0].data.end() - 2, user[0].data.end());
    reduce_action.push_back(mvr::output_reduce_data{mvr::shard_index(2),
            reduce_data, dev::bytes(funcname, funcname + 19)});
    mvr::mvvm_output output(shard_out, out_user, mtv, reduce_action);
    auto bs = mvr::rlp_encode_from_mvvm_output(output);
    /// End function here

    dev::bytes * res = new dev::bytes(bs.begin(), bs.end());
    return res;
}

extern dev::bytes * reduce_exec(byte * bytes, int size) {
    dev::bytesConstRef bcr(bytes, size);

    /// Start function here
    auto input = mvr::rlp_decode_to_mvvm_input(bcr);
    auto shard = input->get_shard_index();
    auto caller = input->get_caller_address();
    auto val = input->get_mtv_value();
    auto shard_data = input->get_input_shard_data();
    auto user = input->get_input_user_data();

    // Reduce action, just add shard data here.

    dev::bytes shard_out(shard_data.begin(), shard_data.end());
    for (auto u : user) for (auto b : u.data)
    { shard_out.push_back(b); }
    vector<mvr::output_user_data> out_user;
    vector<mvr::output_mtv_data> mtv;
    vector<mvr::output_reduce_data> reduce_action;

    mvr::mvvm_output output(shard_out, out_user, mtv, reduce_action);
    auto bs = mvr::rlp_encode_from_mvvm_output(output);
    /// End function here

    dev::bytes * res = new dev::bytes(bs.begin(), bs.end());
    return res;
}

int main()
{
    byte bs[] = {
        byte(248), byte(153), byte(154), byte(97), byte(98), byte(99), byte(100), byte(101), byte(102), byte(103),
        byte(104), byte(105), byte(106), byte(107), byte(108), byte(109), byte(110), byte(111), byte(112), byte(113),
        byte(114), byte(115), byte(116), byte(117), byte(118), byte(119), byte(120), byte(121), byte(122), byte(1),
        byte(164), byte(77), byte(84), byte(86), byte(81), byte(76), byte(98), byte(122), byte(55), byte(74), byte(72),
        byte(105), byte(66), byte(84), byte(115), byte(112), byte(83), byte(57), byte(54), byte(50), byte(82), byte(76),
        byte(75), byte(86), byte(56), byte(71), byte(110), byte(100), byte(87), byte(70), byte(119), byte(106),
        byte(65), byte(53), byte(75), byte(54), byte(54), byte(130), byte(3), byte(232), byte(154), byte(97), byte(98),
        byte(99), byte(100), byte(101), byte(102), byte(103), byte(104), byte(105), byte(106), byte(107), byte(108),
        byte(109), byte(110), byte(111), byte(112), byte(113), byte(114), byte(115), byte(116), byte(117), byte(118),
        byte(119), byte(120), byte(121), byte(122), byte(248), byte(56), byte(219), byte(154), byte(97), byte(98),
        byte(99), byte(100), byte(101), byte(102), byte(103), byte(104), byte(105), byte(106), byte(107), byte(108),
        byte(109), byte(110), byte(111), byte(112), byte(113), byte(114), byte(115), byte(116), byte(117), byte(118),
        byte(119), byte(120), byte(121), byte(122), byte(219), byte(154), byte(97), byte(98), byte(99), byte(100),
        byte(101), byte(102), byte(103), byte(104), byte(105), byte(106), byte(107), byte(108), byte(109), byte(110),
        byte(111), byte(112), byte(113), byte(114), byte(115), byte(116), byte(117), byte(118), byte(119), byte(120),
        byte(121), byte(122)
    };

    auto bytes = exec(bs, sizeof(bs) / sizeof(byte));
    printf("-------- RESULT --------\n");
    for (auto b : *bytes) { printf("%d ", int(b)); } printf("\n");
    delete bytes;
    bytes = reduce_exec(bs, sizeof(bs) / sizeof(byte));
    printf("-------- REDUCE RESULT --------\n");
    for (auto b : *bytes) { printf("%d ", int(b)); } printf("\n");
    delete bytes;
    return 0;
}
