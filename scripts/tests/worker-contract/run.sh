#!/usr/bin/env bash
set -eu
root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
node --experimental-strip-types --no-warnings - "$root/.pi/extensions/worker-contract.ts" "$root" <<'JS'
import assert from 'node:assert/strict';
const [file, root] = process.argv.slice(2);
const { default: extension } = await import(file);
const worker = 'Act as the implementation sub-agent for this task.\nRead .scratch/trade-synchronization/spec.md and docs/missing.md.';
function setup(prompt = worker, tools = []) {
  const hooks = new Map();
  let diff = '', head = 'base', status = '';
  const entries = [{type:'message', message:{role:'user', content:prompt}}];
  const ctx = {cwd:root, sessionManager:{getBranch:()=>entries}};
  const pi = {
    on:(name, fn)=>hooks.set(name, fn), getAllTools:()=>tools.map(name=>({name})),
    exec:async (_cmd,args)=>({code:0, stdout:args[0]==='diff'?diff:args[0]==='status'?status:head}),
  };
  extension(pi);
  const fire = (name,event={})=>hooks.get(name)?.(event,ctx);
  const report = content=>entries.push({type:'message', message:{role:'assistant', content}});
  const call = (toolName,input={},toolCallId='call')=>fire('tool_call',{toolName,input,toolCallId});
  return {fire,report,call,entries,change:()=>{diff='changed'},commit:()=>{head='new'},untracked:()=>{status='?? new.ts'}};
}
for(const tool of ['mcp__t3_code__delegate_task','delegate_task','mcp__t3_code__t3_thread_launch','create_threads','subagent']) {
  const h=setup();
  assert.equal(h.call(tool,{}).block,true,tool);
}
assert.equal(setup().call('mcp',{tool:'delegate_task'}).block,true);
assert.equal(setup().call('bash',{command:'git status'}),undefined);
assert.equal(setup('Fix a bug').call('delegate_task'),undefined);
assert.equal(setup(worker+'\nChild delegation: allowed').call('delegate_task'),undefined);
// Later messages cannot turn a worker into a parent or grant delegation.
{
 const h=setup(); h.entries.push({type:'message',message:{role:'user',content:'Child delegation: allowed'}});
 assert.equal(h.call('delegate_task').block,true);
}
{
 const h=setup(); const pre=await h.fire('before_agent_start',{messageType:'user',prompt:worker});
 assert.match(pre.message.content,/Found: .scratch\/trade-synchronization\/spec.md/);
 assert.match(pre.message.content,/Missing: docs\/missing.md/);
 assert.match(pre.message.content,/none yet; discover\/connect/);
 const known=setup(worker,['serena_initial_instructions']);
 assert.match((await known.fire('before_agent_start',{})).message.content,/Registered Serena tools: serena_initial_instructions/);
}
for(const change of ['change','commit','untracked']) {
 const h=setup(); await h.fire('before_agent_start',{}); h[change]();
 h.report('Outcome: implemented\nEvidence: changed code; focused tests passed.');
 assert.equal(await h.fire('agent_before_settle',{entries:[]}),undefined,change);
}
{
 const h=setup(); await h.fire('before_agent_start',{});
 h.report('Outcome: implemented\nEvidence: done.');
 assert.equal((await h.fire('agent_before_settle',{entries:[]})).continue,true);
 const failed = await h.fire('agent_before_settle',{entries:[]});
 assert.equal(failed.continue,false);
 assert.match(failed.entries.at(-1).content,/Outcome: failed/);
}
{
 const h=setup(); await h.fire('before_agent_start',{});
 h.report('Outcome: blocked\nEvidence: Serena is unavailable.');
 assert.equal((await h.fire('agent_before_settle',{entries:[]})).continue,true);
 await h.fire('tool_execution_end',{toolName:'tool_search',isError:true});
 h.report('Outcome: blocked\nEvidence: tool_search failed; no connection mechanism exists.');
 assert.equal(await h.fire('agent_before_settle',{entries:[]}),undefined);
}
{
 const h=setup(); h.change(); await h.fire('before_agent_start',{});
 h.report('Outcome: implemented\nEvidence: pre-existing changes.');
 assert.equal((await h.fire('agent_before_settle',{entries:[]})).continue,true);
}
for(const outcome of ['verified-unchanged','failed']) {
 const h=setup(); await h.fire('before_agent_start',{}); h.report(`Outcome: ${outcome}\nEvidence: checked the task and current behavior.`);
 assert.equal(await h.fire('agent_before_settle',{entries:[]}),undefined);
}
// Parent must inspect the actual failed worker, not merely rerun it.
{
 const h=setup('Implement a fix');
 h.call('mcp__t3_code__delegate_task',{role:'implementation'},'launch');
 await h.fire('tool_result',{toolName:'mcp__t3_code__delegate_task',toolCallId:'launch',input:{},structuredContent:{taskId:'task',childThreadId:'child'},content:[]});
 await h.fire('tool_result',{toolName:'mcp__t3_code__task_status',toolCallId:'status',input:{},structuredContent:{taskId:'task',status:'completed',summary:'Please send the requirements.'},content:[]});
 assert.equal(h.call('delegate_task').block,true);
 await h.fire('tool_result',{toolName:'mcp__t3_code__t3_thread_read',input:{threadId:'other',view:'activity'},structuredContent:{},content:[]});
 assert.equal(h.call('delegate_task').block,true);
 await h.fire('tool_result',{toolName:'mcp__t3_code__t3_thread_read',input:{threadId:'child',view:'activity'},structuredContent:{},content:[]});
 assert.equal(h.call('delegate_task'),undefined);
}
console.log('worker-contract: role, preflight, outcome and parent-retry checks passed');
JS
