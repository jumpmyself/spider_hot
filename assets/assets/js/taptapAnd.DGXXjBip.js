import{E as x,a as E,b as A}from"./hotSpots.DT9Ygqio.js";import{_ as B,g as C,v as D,a as M}from"./axios.SImg6CTL.js";import{_ as T}from"./taptap.Drb_1m4Z.js";import{r as c,l as b,aa as z,t as s,v as e,E as i,H as o,$ as t,S as p,B as F,F as I,a7 as N,z as S}from"./index.Cn214qmR.js";const U={class:"card"},V=t("div",{class:"logo"},[t("img",{src:T,alt:""}),t("span",null,"taptop(安卓)")],-1),H={class:"icon"},$=t("span",null,"热搜榜",-1),j={key:0,class:"cardLoading"},q={key:1,class:"detaList"},G={class:"top"},J={class:"index"},K={class:"title"},O=["href"],P=t("img",{src:B,alt:"获取最新",title:"获取最新"},null,-1),Z={__name:"taptapAnd",setup(Q){const u=c([]),v=c(null),r=c(""),a=c(!0),k=async(h,g,n,d)=>{const l=await M.get(h);console.log(l),g.value=l.data.data,n.value=new Date,d.value=C(n.value),a.value=!1},m=async()=>{await k("https://sss.spider-hot.top/taptapand",u,v,r),a.value=!0,setTimeout(()=>{a.value=!1},500)},w=()=>{setInterval(()=>{r.value=C(v.value)},1e3)};return b(()=>{m(),w()}),(h,g)=>{const n=z("CircleCheck"),d=x,l=E,y=A,L=D;return s(),e("div",U,[i(y,{style:{width:"409px",margin:"10px"},shadow:"always"},{header:o(()=>[V,t("div",H,[i(d,null,{default:o(()=>[i(n)]),_:1}),$])]),footer:o(()=>[t("p",null,p(r.value),1),i(l,{onClick:m},{default:o(()=>[P]),_:1})]),default:o(()=>[t("div",null,[a.value?F((s(),e("div",j,null,512)),[[L,a.value]]):(s(),e("div",q,[(s(!0),e(I,null,N(u.value,(f,_)=>(s(),e("p",{key:_,class:S(["text","item"+_])},[t("div",G,[t("span",J,p(_+1),1),t("div",K,[t("a",{href:f.url},p(f.title),9,O)])])],2))),128))]))])]),_:1})])}}};export{Z as default};
