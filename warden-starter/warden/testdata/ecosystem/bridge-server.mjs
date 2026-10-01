// Disposable stdio fixture: no dependencies, network calls or credentials.
import readline from "node:readline";
import fs from "node:fs/promises";
const input=readline.createInterface({input:process.stdin});
const tools=["read","write","erase"].map(name=>({name,description:`Fixture ${name}`,inputSchema:{type:"object",properties:{path:{type:"string"},content:{type:"string"}},required:["path"]}}));
input.on("line",async line=>{const m=JSON.parse(line);if(m.id===undefined)return;let result={};let error;
 switch(m.method){
 case "initialize":result={protocolVersion:"2025-11-25",capabilities:{tools:{}},serverInfo:{name:"warden-disposable-fixture",version:"1"}};break;
 case "ping":break;
 case "tools/list":result={tools};break;
 case "tools/call":try{const a=m.params.arguments;if(m.params.name==="read")result={content:[{type:"text",text:await fs.readFile(a.path,"utf8")}]};else if(m.params.name==="write"){await fs.writeFile(a.path,a.content);result={content:[{type:"text",text:"written"}]}}else if(m.params.name==="erase"){await fs.unlink(a.path);result={content:[]}}else error={code:-32601,message:"unknown tool"}}catch{result={isError:true,content:[{type:"text",text:"filesystem denied"}]}}break;
 default:error={code:-32601,message:"method unsupported"};
 }
 process.stdout.write(JSON.stringify({jsonrpc:"2.0",id:m.id,...(error?{error}:{result})})+"\n");
});
