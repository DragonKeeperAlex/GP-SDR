// Run: node Scripts/test_audio_queue.cjs
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const script = fs.readFileSync(require('node:path').join(__dirname, '../server/web/app.js'), 'utf8');
const begin = script.indexOf('function clearScheduledAudio(');
const end = script.indexOf('async function startLiveAudio()', begin);
const made = [];
const context = {
  currentTime: 10,
  createBuffer: (_, count, rate) => ({duration: count / rate, getChannelData: () => new Float32Array(count)}),
  createBufferSource: () => {
    const source = {connect() {}, disconnect() {this.disconnected = true;}, stop() {this.stopped = true;}, start(time) {this.time = time;}};
    made.push(source); return source;
  }
};
const liveAudio = {context, nextTimes: new Map(), sources: new Map()};
const sandbox = {liveAudio, channelGain: () => ({})};
vm.createContext(sandbox);
vm.runInContext(script.slice(begin, end), sandbox);
const pcm = new Int16Array(160);
sandbox.scheduleAudioFrame('one', 8000, pcm);
sandbox.scheduleAudioFrame('one', 8000, pcm);
assert.equal(made[0].time, 10.06);
assert.equal(made[1].time, 10.08);
sandbox.scheduleAudioFrame('two', 8000, pcm);
liveAudio.nextTimes.set('one', 11);
sandbox.scheduleAudioFrame('one', 8000, pcm);
assert.ok(made[0].stopped && made[1].stopped);
assert.ok(!made[2].stopped, 'reset must not cut another channel');
assert.equal(made[3].time, 10.06);
made[3].onended();
assert.ok(made[3].disconnected);
assert.ok(!liveAudio.sources.has('one'));
sandbox.clearScheduledAudio();
assert.ok(made[2].stopped && made[2].disconnected);
assert.equal(liveAudio.sources.size, 0);
console.log('Audio queue continuity, isolated backlog reset, completion and stop: PASS');
