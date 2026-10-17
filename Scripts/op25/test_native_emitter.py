#!/usr/bin/env python3
"""Validate the linked emitter in tagged and upstream-compatible legacy modes."""
import os,socket,subprocess,sys,struct
for tagged in (False,True):
    with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as receiver:
        receiver.bind(('127.0.0.1',0));receiver.settimeout(3)
        env={**os.environ,'GPSDR_P25_TAGGED_AUDIO':'1' if tagged else '0'}
        subprocess.run([sys.argv[1],str(receiver.getsockname()[1])],env=env,check=True,stdout=subprocess.DEVNULL)
        packets=[receiver.recv(4096) for _ in range(4)]
        if tagged:
            assert all(p[:8]==b'GPSDP25\0' and p[8]==1 for p in packets)
            assert [struct.unpack_from('<I',p,12)[0] for p in packets]==[1,2,3,4]
            assert [struct.unpack_from('<I',p,20)[0] for p in packets]==[100,101,101,102]
            assert packets[2][9]==1 and len(packets[2])==32
            assert packets[3][10]&4 and packets[3][31]==0x84
            assert packets[0][32:]==b'\x01\0\xfe\xff'
        else:
            assert packets==[b'\x01\0\xfe\xff',b'\x01\0\xfe\xff',b'\0\0',b'\x01\0\xfe\xff']
print('Linked OP25 emitter: tagged identity/sequence/crypto and legacy wire compatibility passed.')
