import fs from 'fs';

let content = fs.readFileSync('dashboard/src/App.tsx', 'utf-8');

const glassCardRegex = /backdrop-blur-\[40px\].*?overflow-hidden/g;
content = content.replace(glassCardRegex, "glass-card");

const glassInputRegex = /bg-gradient-to-br from-white\/\[0\.04\].*?focus:from-white\/\[0\.07\] focus:to-white\/\[0\.03\]/g;
content = content.replace(glassInputRegex, "glass-input text-sm text-slate-200 outline-none backdrop-blur-xl");

const glassBtnRegex = /bg-gradient-to-br from-indigo-500\/90 to-purple-600\/90.*?active:scale-\[1\.01\]/g;
content = content.replace(glassBtnRegex, "glass-btn text-white");

const glassInnerRegex = /bg-gradient-to-br from-white\/\[0\.038\].*?hover:border-white\/\[0\.1\]/g;
content = content.replace(glassInnerRegex, "glass-inner p-5 flex flex-col gap-3");

const glassHoverItemRegex = /hover:scale-\[1\.02\].*?focus:to-white\/\[0\.03\]/g;
content = content.replace(glassHoverItemRegex, "glass-inner");

// The raw stream container might have a leftover bg-white/5 string
content = content.replace(/bg-white\/5 backdrop-blur-3xl border border-white\/10 rounded-\[2rem\] shadow-2xl/g, "glass-card");

// Drop some of the absolute positions logic if we used glass-card which is mostly relative anyway
content = content.replace(/glass-card px-8 py-5 relative/g, "glass-card px-8 py-5");
content = content.replace(/glass-card p-8 relative flex/g, "glass-card p-8 flex");
content = content.replace(/glass-card p-8 relative/g, "glass-card p-8");

fs.writeFileSync('dashboard/src/App.tsx', content);

