#!/usr/bin/env python3
"""Apply the reviewed OP25 patch only to the exact inspected source files.
Run cmake/build/installation separately. Does not modify radio configuration.
"""
import argparse, hashlib, json, pathlib, shutil, subprocess
parser=argparse.ArgumentParser()
parser.add_argument('--source',required=True,type=pathlib.Path)
args=parser.parse_args()
here=pathlib.Path(__file__).resolve().parent
library=args.source/'op25/gr-op25_repeater/lib'
manifest=json.loads((here/'native-audio-sources.json').read_text())
states=[]
for name,expected in manifest.items():
    digest=hashlib.sha256((library/name).read_text().encode()).hexdigest()
    state=next((kind for kind,want in expected.items() if digest==want),None)
    if state is None: raise SystemExit(f'{name}: unrecognized source; preserve changes and review before applying')
    states.append(state)
header=library/'gpsdr_audio_tag.h'
if header.exists() and header.read_bytes()!=(here/header.name).read_bytes():
    raise SystemExit('Existing native-audio header differs; preserve and review it')
if all(s=='original' for s in states):
    subprocess.run(['git','-C',str(args.source),'apply','--check','--ignore-space-change',str(here/'native-audio.patch')],check=True)
    subprocess.run(['git','-C',str(args.source),'apply','--ignore-space-change',str(here/'native-audio.patch')],check=True)
elif not all(s=='patched' for s in states):
    raise SystemExit('Partially applied source; preserve and review it before continuing')
if not header.exists(): shutil.copy2(here/header.name,header)
for name,expected in manifest.items():
    assert hashlib.sha256((library/name).read_text().encode()).hexdigest()==expected['patched'],name
print('Reviewed native-audio source patch ready; build and test before installation.')
