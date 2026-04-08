import fs from 'fs';
let content = fs.readFileSync('dashboard/src/App.tsx', 'utf-8');

content = content.replace(/bg-\[#050505\] text-slate-100 font-sans p-6 overflow-x-hidden relative/g, 'bg-black text-slate-200 font-sans p-4 md:p-8 overflow-x-hidden relative selection:bg-blue-500/30');

content = content.replace(/bg-blue-600\/10 blur-\[150px\]/g, 'bg-blue-900/10 blur-[120px] rounded-full');
content = content.replace(/bg-purple-600\/10 blur-\[150px\]/g, 'bg-indigo-900/10 blur-[120px] rounded-full');
content = content.replace(/w-\[40%\] h-\[40%\]/g, 'w-[50%] h-[50%]');

content = content.replace(/mb-8 pb-4 border-b border-white\/10 bg-white\/5 backdrop-blur-xl px-6 py-4 rounded-2xl shadow-2xl/g, 'mb-8 border border-white/[0.05] bg-white/[0.02] backdrop-blur-2xl px-6 py-4 rounded-3xl shadow-[0_8px_32px_0_rgba(0,0,0,0.5)]');

content = content.replace(/<Shield size=\{32\} className="text-blue-400" \/>/g, '<Shield size={32} className="text-blue-400 drop-shadow-[0_0_15px_rgba(96,165,250,0.5)]" />');

content = content.replace(/bg-white\/5 backdrop-blur-xl border border-white\/10 rounded-2xl p-6 shadow-2xl/g, 'bg-white/[0.02] backdrop-blur-2xl border border-white/[0.05] rounded-3xl p-6 shadow-[0_8px_32px_0_rgba(0,0,0,0.5)]');

content = content.replace(/text-xs uppercase tracking-widest text-slate-400 font-bold flex items-center gap-2/g, 'text-[10px] uppercase tracking-widest text-slate-400 font-semibold flex items-center gap-2 ml-1');

content = content.replace(/w-full bg-black\/40 border border-white\/10 rounded-lg px-4 py-3 text-white outline-none focus:border-blue-500 transition-colors backdrop-blur-md/g, 'w-full bg-black/60 border border-white/10 rounded-xl px-4 py-3.5 text-white outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 transition-all backdrop-blur-xl appearance-none');

content = content.replace(/bg-blue-600 hover:bg-blue-500 text-white font-bold px-8 py-3 rounded-lg transition-all shadow-\[0_0_20px_rgba\(37,99,235,0\.4\)\] hover:shadow-\[0_0_30px_rgba\(37,99,235,0\.6\)\]/g, 'bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-bold px-8 py-3.5 rounded-xl transition-all shadow-[0_0_30px_rgba(37,99,235,0.3)] hover:shadow-[0_0_40px_rgba(37,99,235,0.6)] border border-white/10');

content = content.replace(/bg-white\/5 backdrop-blur-xl border border-white\/10 p-6 rounded-2xl flex/g, 'bg-white/[0.02] backdrop-blur-2xl border border-white/[0.05] p-6 rounded-3xl flex');

content = content.replace(/text-slate-400 text-xs uppercase tracking-widest mb-1 font-bold/g, 'text-slate-500 text-[10px] uppercase tracking-widest mb-2 font-semibold');

content = content.replace(/text-4xl font-light text-white/g, 'text-4xl font-light text-white tracking-tight');

content = content.replace(/text-sm uppercase tracking-widest font-bold text-slate-300 mb-6 flex items-center gap-2/g, 'text-[10px] uppercase tracking-widest font-semibold text-slate-400 mb-6 flex items-center gap-2');
content = content.replace(/text-sm uppercase tracking-widest font-bold text-slate-300 mb-4 flex items-center gap-2/g, 'text-[10px] uppercase tracking-widest font-semibold text-slate-400 mb-4 flex items-center gap-2');

content = content.replace(/rgba\(15,23,42,0\.9\)/g, 'rgba(0,0,0,0.8)');
content = content.replace(/blur\(10px\)/g, 'blur(16px)');
content = content.replace(/rgba\(255,255,255,0\.1\)/g, 'rgba(255,255,255,0.05)');
content = content.replace(/borderRadius: "8px"/g, 'borderRadius: "12px", boxShadow: "0 4px 24px rgba(0,0,0,0.5)"');

content = content.replace(/bg-black\/40 backdrop-blur-md p-3 rounded-xl border border-white\/5/g, 'bg-black/40 backdrop-blur-xl p-4 rounded-2xl border border-white/[0.05] hover:bg-black/60 transition-colors');

content = content.replace(/font-mono text-sm text-blue-300 drop-shadow-\[0_0_5px_rgba\(147,197,253,0\.5\)\]/g, 'font-mono text-sm text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.3)]');

content = content.replace(/text-xs text-slate-400 truncate/g, 'text-[11px] text-slate-500 truncate leading-relaxed');

content = content.replace(/bg-black\/60 px-6 py-4 border-b border-white\/10 flex items-center gap-3 relative/g, 'bg-black/40 px-6 py-5 border-b border-white/5 flex items-center gap-3 relative backdrop-blur-xl');

content = content.replace(/p-6 h-64 overflow-y-auto font-mono text-xs space-y-1 custom-scrollbar text-slate-400/g, 'p-6 h-72 overflow-y-auto font-mono text-[11px] space-y-1.5 custom-scrollbar text-slate-500 relative z-10');

content = content.replace(/sm:gap-6 hover:bg-white\/5 py-1\.5 px-3 rounded-lg transition-colors border border-transparent hover:border-white\/5/g, 'sm:gap-6 hover:bg-white/[0.03] py-2 px-4 rounded-xl transition-all border border-transparent hover:border-white/[0.05]');

content = content.replace(/opacity-50 shrink-0 w-24/g, 'text-slate-600 shrink-0 w-24');
content = content.replace(/text-blue-300 shrink-0 w-28 drop-shadow-\[0_0_3px_rgba\(147,197,253,0\.3\)\]/g, 'text-blue-400/80 shrink-0 w-28');
content = content.replace(/text-indigo-400 shrink-0 w-36 opacity-80/g, 'text-indigo-400/70 shrink-0 w-36');
content = content.replace(/text-slate-300 flex-1 truncate/g, 'text-slate-400 flex-1 truncate');
content = content.replace(/<span className="opacity-50">→<\/span>/g, '<span className="text-slate-700 mx-2">→</span>');

content = content.replace(/width: 4px;/g, 'width: 5px;');

fs.writeFileSync('dashboard/src/App.tsx', content);

