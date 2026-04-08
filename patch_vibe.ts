import fs from 'fs';
let content = fs.readFileSync('dashboard/src/App.tsx', 'utf-8');

// 1. Root background
content = content.replace(/bg-\[#070709\]/g, "bg-black");

// 2. Base abstract glows
content = content.replace(/bg-blue-500\/5 blur-\[150px\]/g, "bg-indigo-500/5 blur-[150px]");
content = content.replace(/bg-emerald-500\/5 blur-\[150px\]/g, "bg-purple-500/5 blur-[150px]");

// 3. The "Liquid Glass Card" formula
const glassCard = "backdrop-blur-[40px] backdrop-saturate-[160%] bg-gradient-to-br from-white/[0.045] via-white/[0.018] to-white/[0.032] border border-white/[0.09] shadow-[0_8px_32px_rgba(0,0,0,0.35),inset_0_0_0_1px_rgba(255,255,255,0.03),0_32px_64px_-12px_rgba(0,0,0,0.45)] rounded-[32px] transition-all duration-[600ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:-translate-y-1 hover:shadow-[0_20px_50px_rgba(0,0,0,0.4),inset_0_0_0_1px_rgba(255,255,255,0.07),0_40px_80px_-20px_rgba(99,102,241,0.12)] hover:border-white/[0.12] overflow-hidden";

// Replace main container classes
const oldContainerSearch = /bg-white\/5 backdrop-blur-3xl border border-white\/10 .*? shadow-2xl/g;
// We'll surgically replace the specific blocks.
content = content.replace(/bg-white\/5 backdrop-blur-3xl border border-white\/10 px-8 py-5 rounded-\[2rem\] shadow-2xl/g, `${glassCard} px-8 py-5 relative`);
content = content.replace(/bg-white\/5 backdrop-blur-3xl border border-white\/10 rounded-\[2rem\] shadow-2xl p-8/g, `${glassCard} p-8 relative`);
content = content.replace(/bg-white\/5 backdrop-blur-3xl border border-white\/10 rounded-\[2rem\] p-8/g, `${glassCard} p-8 relative`);
content = content.replace(/bg-white\/5 backdrop-blur-3xl border border-white\/10 p-8 rounded-\[2rem\] shadow-2xl/g, `${glassCard} p-8 relative`);

// 4. Inputs
content = content.replace(/bg-black\/20 border border-white\/10 rounded-2xl px-5 py-4/g, "bg-gradient-to-br from-white/[0.04] to-white/[0.018] border border-white/[0.07] rounded-[20px] px-5 py-4 shadow-[0_8px_32px_rgba(99,102,241,0.1),inset_0_0_0_1px_rgba(255,255,255,0.06)]");
// Input hover/focus states
content = content.replace(/hover:bg-white\/5/g, "hover:scale-[1.02] hover:border-white/[0.12] focus:scale-[1.02] focus:border-white/[0.12] focus:bg-gradient-to-br focus:from-white/[0.07] focus:to-white/[0.03]");

// 5. Button
const buttonClass = "bg-gradient-to-br from-indigo-500/90 to-purple-600/90 text-white shadow-[0_4px_15px_rgba(99,102,241,0.3),inset_0_1px_0_rgba(255,255,255,0.2)] rounded-[14px] transition-all duration-400 ease-[cubic-bezier(0.23,1,0.32,1)] hover:-translate-y-[3px] hover:scale-[1.02] hover:shadow-[0_12px_35px_rgba(99,102,241,0.4),0_4px_15px_rgba(139,92,246,0.3),inset_0_1px_0_rgba(255,255,255,0.3)] active:translate-y-[-1px] active:scale-[1.01]";
content = content.replace(/bg-white text-black hover:bg-slate-200 .*? rounded-2xl transition-all shadow-\[0_4px_20px_rgba\(255,255,255,0\.1\)\] hover:shadow-\[0_8px_30px_rgba\(255,255,255,0\.2\)\] active:scale-95/g, buttonClass);

// Inner Alert elements
content = content.replace(/bg-white\/5 p-5 rounded-2xl border border-white\/5 flex flex-col gap-3 transition-colors hover:bg-white\/10/g, "bg-gradient-to-br from-white/[0.038] to-white/[0.012] border border-white/[0.07] rounded-[18px] p-5 flex flex-col gap-3 transition-all duration-[400ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-gradient-to-br hover:from-white/[0.055] hover:to-white/[0.022] hover:border-white/[0.1]");

fs.writeFileSync('dashboard/src/App.tsx', content);
