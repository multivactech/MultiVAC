#include "mvvm_inout.h"

using namespace std;

mvr::mvvm_input * mvr::rlp_decode_to_mvvm_input(dev::bytesConstRef bytes)
{
    vector<byte> params, shard_data, address;
    mvr::shard_index shard; long value;
    vector<mvr::input_user_data> user_data;

    dev::RLP rlp(bytes);
    if (rlp.isNull() || rlp.isEmpty() || rlp.itemCountStrict() != mvr::INPUT_ITEM_SIZE)
    { BOOST_THROW_EXCEPTION(dev::BadRLP()); return new mvvm_input(); }

    for (auto i = 0; i < mvr::INPUT_ITEM_SIZE; i++) {
        auto field = mvr::mvvm_input_fields(i);
        switch (field) {
        case FIELD_INPUT_PARAMS:
            params = vector<byte>(rlp[field]); break;
        case FIELD_INPUT_SHARD_INDEX:
            shard = rlp[field].toInt(); break;
        case FIELD_INPUT_ADDRESS:
            address = vector<byte>(rlp[field]); break;
        case FIELD_INPUT_MTV_VALUE:
            value = rlp[field].toPositiveInt64(); break;
        case FIELD_INPUT_SHARD_DATA:
            shard_data = vector<byte>(rlp[field]); break;
        case FIELD_INPUT_USER_DATA:
            auto iCnt = rlp[field].itemCountStrict();
            for (auto j = 0; j < iCnt; j++) {
                if (rlp[field][j].isNull() || rlp[field][j].isEmpty() || rlp[field][j].itemCountStrict() != 1)
                { BOOST_THROW_EXCEPTION(dev::BadRLP()); return new mvvm_input(); }
                user_data.push_back(mvr::input_user_data{std::vector<byte>(rlp[field][j][0])});
            }
            break;
        }
    }
    
    mvvm_input * input = new mvvm_input(params, shard, address, value, shard_data, user_data);
    return input;
}

dev::bytes mvr::rlp_encode_from_mvvm_output(const mvr::mvvm_output & output)
{
    dev::RLPStream stream, stream_shard_data, stream_user_data, stream_mtv_data, stream_reduce_data;
    auto user_data = output.get_output_user_data();
    auto mtv_data = output.get_output_mtv_data();
    auto reduce_data = output.get_output_reduce_data();

    for (auto ud : user_data) {
        dev::RLPStream s;
        s << ud.shard << ud.useraddress << ud.data;
        stream_user_data.appendList(s);
    }
    for (auto md : mtv_data) {
        dev::RLPStream s;
        s << md.shard << md.useraddress << md.value;
        stream_mtv_data.appendList(s);
    }
    for (auto rd : reduce_data) {
        dev::RLPStream s;
        s << rd.shard << rd.data << rd.api;
        stream_reduce_data.appendList(s);
    }

    dev::RLPStream s; s << output.get_output_shard_data();
    s.appendList(stream_user_data); s.appendList(stream_mtv_data); s.appendList(stream_reduce_data);
    stream.appendList(s);

    return stream.out();
}
