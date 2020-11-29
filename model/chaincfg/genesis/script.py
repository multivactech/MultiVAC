import sys
import ed25519
import numpy as np

ispython3 = sys.version.startswith('3')

def generate_shard_params(prefix_length):
  template = open('shard.go.template').read()
  params = template % {'prefix_length' : prefix_length}
  open('../../params/shard.go', 'w').write(params)

def generate_private_keys(prefix_length):
  keys = {}
  number_of_keys = (1 << prefix_length)
  found_keys = 0
  mask = ~np.uint32((1 << (32-prefix_length)) -1)
  while True:
    private_key, public_key = ed25519.keys.create_keypair()
    pk_prefix_bytes = public_key.to_bytes()[0:4]
    pk_prefix = np.uint32(0)
    for i in pk_prefix_bytes:
      pk_prefix = pk_prefix << 8
      if ispython3:
        pk_prefix += i
      else:
        pk_prefix += ord(i)
    pk_prefix = pk_prefix & mask
    if pk_prefix not in keys:
      keys[pk_prefix] = private_key.to_bytes()
      found_keys += 1
      if found_keys == number_of_keys:
        break
  keys_template = '''\tkeys[ShardList[%(index)s]] = signature.PrivateKey{%(key_bytes)s\n\t}\n'''
  keys_str = ""
  for k in range(number_of_keys):
    privkey_bytes = keys[np.uint32(k) << (32-prefix_length)]
    key_bytes = ""
    for i in range(len(privkey_bytes)):
      if i % 8 == 0:
        key_bytes += "\n\t\t"
      else:
        key_bytes += " "
      if ispython3:
        key_bytes += "%s," % privkey_bytes[i]
      else:
        key_bytes += "%s," % ord(privkey_bytes[i])
    keys_str += keys_template % {'index': k, 'key_bytes': key_bytes}
  template = open('privatekeys.go.template').read()
  config = template % {'keys_str' : keys_str}
  open('generated_privatekeys.go', 'w').write(config)

if __name__ == "__main__":
  if len(sys.argv) < 2:
    print('''usage: python %s <prefix_length>
e.g. To create genesis blocks and keys for 128 shards, run
  python %s 7
note 2^7=128''' % (sys.argv[0], sys.argv[0]))
    sys.exit()

  prefix_length = int(sys.argv[1])
  generate_shard_params(prefix_length)
  generate_private_keys(prefix_length)
