window.OnlineWebFonts_Animations=window.OnlineWebFonts_Animations||function(t){return"object"!=typeof t?this:{Config:{},Index:{Key:"www.onlinewebfonts.com",Id:null,Data:{Config:{Width:3,Opacity:1,StrokeDot:!0,Sequential:!0,Display:!0,Reverse:!1,Color:"#000000",Animate:"Linear"}},Svg:{},Path:[],Div:null,An:null,Dom:null,Pause:!1,Complete:null,Status:null},Run:function(t){this.Config=this.Index;var n=this.Config,e=n.Data;for(var i in e.Config)null!=t.Data.Config[i]&&(e.Config[i]=t.Data.Config[i]);return n.Id=t.Id,n.Data.Line=t.Data.Line,n.Data.Box=t.Data.Box,"function"==typeof t.Complete&&(n.Complete=t.Complete),"function"==typeof t.Status&&(n.Status=t.Status),this.Append().PathAppend(),this},Play:function(){var t=this;return t.Stop(),t.Config.An=requestAnimationFrame((function(n){t.Update(n)})),this},Pause:function(){return this.Config.Pause||(this.Config.Pause=!0,cancelAnimationFrame(this.Config.An)),this},Resume:function(){var t=this;return t.Config.Pause&&(t.Config.Pause=!1,requestAnimationFrame((function(n){t.ResumeUpdate(n)}))),this},Stop:function(){return this.Config.Div.innerHTML="",this.Append().PathAppend(),cancelAnimationFrame(this.Config.An),this},ResumeUpdate:function(t){var n=this,e=n.Config.Svg.Time.Data;e.Start=t-e.Elapsed,requestAnimationFrame((function(t){n.Update(t)}))},Update:function(t){var n=this,e=n.Config,i=e.Data,r=e.Svg.Time.Data;if(0==r.Start&&(r.Start=t),r.Elapsed=t-r.Start,r.Progress=n.Progress(r.Total,r.Start,r.Elapsed,i.Config.Animate),n.UpdatePath(),r.Progress<1){if(null!==e.Status){var a=Math.round(100*r.Progress);e.Status(99==a?100:a,e.Id)}n.Config.An=requestAnimationFrame((function(t){n.Update(t)}))}else null!==e.Complete&&e.Complete()},UpdatePath:function(){for(var t=this.Config.Svg.Time.Path,n=0;n<this.Config.Dom.length;n++){var e=this.PathElapsed(n);t[n].Progress=this.Progress(1,0,e,this.Config.Data.Config.Animate),this.SetLine(n)}},SetLine:function(t){var n=this.Config.Svg,e=n.Time.Path,i=this.Config.Dom,r=e[t].Progress*n.Path.Length[t];if(this.Config.Data.Config.Reverse)var a=-n.Path.Length[t]+r;else a=n.Path.Length[t]-r;i[t].style.strokeDashoffset=a},PathElapsed:function(t){var n,e=this.Config.Svg.Time,i=e.Path[t];return e.Data.Progress>i.StartPro&&e.Data.Progress<i.StartPro+i.Duration?n=(e.Data.Progress-i.StartPro)/i.Duration:e.Data.Progress>=i.StartPro+i.Duration?n=1:e.Data.Progress<=i.StartPro&&(n=0),n},Progress:function(t,n,e,i){var r;return e>0&&e<t?r=i?this.Ease[i](e,0,1,t):e/t:e>=t?r=1:e<=n&&(r=0),r},PathAppend:function(){var t=this.Config,n=t.Data,e=t.Svg.Time;e.Path=[];var i=n.Config.Reverse?e.Data.Total:0;for(var r in n.Line){var a=parseInt(n.Line[r].Duration),o=a/e.Data.Total;n.Config.Reverse?i-=a:i=n.Config.Sequential?e.Data.Delay:0;var u=i/e.Data.Total;e.Data.Delay+=a,e.Path[r]={Start:i,Duration:o,StartPro:u}}},Append:function(){var t=this.Config,n=t.Data,e=t.Svg,i=this.SVGElement();e.Path={},e.Time={},e.Time.Data={Start:0,Elapsed:0,Total:0,Duration:0,Progress:0,Delay:0},e.Path.Length=[];var r=0;for(var a in n.Line){var o={"fill-opacity":"0","stroke-linecap":n.Config.StrokeDot?"round":"butt","stroke-linejoin":"round",stroke:n.Line[a].Color?n.Line[a].Color:n.Config.Color,"stroke-opacity":n.Config.StrokeDot?n.Config.Opacity:"0","stroke-width":n.Line[a].Width?n.Line[a].Width:n.Config.Width,d:n.Line[a].Path},u=document.createElementNS("http://www.w3.org/2000/svg","path");for(var s in o)u.setAttribute(s,o[s]);var f=Math.ceil(u.getTotalLength());e.Path.Length[a]=f,u.setAttribute("style","stroke-dasharray:"+f+","+f+";stroke-dashoffset:"+(n.Config.Display?"0":f)+";"),i.appendChild(u),t.Path.push(u),0==n.Line[a].Duration&&(n.Line[a].Duration=this.GetPathDuration(n.Line[a].Path)),n.Config.Sequential?r+=parseInt(n.Line[a].Duration):parseInt(n.Line[a].Duration)>r&&(r=parseInt(n.Line[a].Duration))}return e.Time.Data.Total=r,t.Div=document.querySelector(t.Id),t.Div.appendChild(i),t.Dom=t.Div.children[0].childNodes,this},GetPathDuration:function(t){var n=document.createElementNS("http://www.w3.org/2000/svg","path");return n.setAttribute("d",t),Math.ceil(n.getTotalLength())},SVGElement:function(t){var n=document.createElementNS("http://www.w3.org/2000/svg","svg");n.setAttribute("xmlns","http://www.w3.org/2000/svg");var e=this.Config.Data.Box.Width,i=this.Config.Data.Box.Height;return n.setAttribute("viewBox","0 0 "+e+" "+i),n},Ease:{Linear:function(t,n,e,i){return e*t/i+n},InQuad:function(t,n,e,i){return e*(t/=i)*t+n},OutQuad:function(t,n,e,i){return-e*(t/=i)*(t-2)+n},InOutQuad:function(t,n,e,i){return(t/=i/2)<1?e/2*t*t+n:-e/2*(--t*(t-2)-1)+n},InCubic:function(t,n,e,i){return e*(t/=i)*t*t+n},OutCubic:function(t,n,e,i){return e*((t=t/i-1)*t*t+1)+n},InOutCubic:function(t,n,e,i){return(t/=i/2)<1?e/2*t*t*t+n:e/2*((t-=2)*t*t+2)+n},InQuart:function(t,n,e,i){return e*(t/=i)*t*t*t+n},OutQuart:function(t,n,e,i){return-e*((t=t/i-1)*t*t*t-1)+n},InOutQuart:function(t,n,e,i){return(t/=i/2)<1?e/2*t*t*t*t+n:-e/2*((t-=2)*t*t*t-2)+n},InQuint:function(t,n,e,i){return e*(t/=i)*t*t*t*t+n},OutQuint:function(t,n,e,i){return e*((t=t/i-1)*t*t*t*t+1)+n},InOutQuint:function(t,n,e,i){return(t/=i/2)<1?e/2*t*t*t*t*t+n:e/2*((t-=2)*t*t*t*t+2)+n},InSine:function(t,n,e,i){return-e*Math.cos(t/i*(Math.PI/2))+e+n},OutSine:function(t,n,e,i){return e*Math.sin(t/i*(Math.PI/2))+n},InOutSine:function(t,n,e,i){return-e/2*(Math.cos(Math.PI*t/i)-1)+n},InExpo:function(t,n,e,i){return 0==t?n:e*Math.pow(2,10*(t/i-1))+n},OutExpo:function(t,n,e,i){return t==i?n+e:e*(1-Math.pow(2,-10*t/i))+n},InOutExpo:function(t,n,e,i){return 0==t?n:t==i?n+e:(t/=i/2)<1?e/2*Math.pow(2,10*(t-1))+n:e/2*(2-Math.pow(2,-10*--t))+n},InCirc:function(t,n,e,i){return-e*(Math.sqrt(1-(t/=i)*t)-1)+n},OutCirc:function(t,n,e,i){return e*Math.sqrt(1-(t=t/i-1)*t)+n},InOutCirc:function(t,n,e,i){return(t/=i/2)<1?-e/2*(Math.sqrt(1-t*t)-1)+n:e/2*(Math.sqrt(1-(t-=2)*t)+1)+n},InElastic:function(t,n,e,i){var r=1.70158,a=0,o=e;if(0==t)return n;if(1==(t/=i))return n+e;if(a||(a=.3*i),o<Math.abs(e)){o=e;r=a/4}else r=a/(2*Math.PI)*Math.asin(e/o);return-o*Math.pow(2,10*(t-=1))*Math.sin((t*i-r)*(2*Math.PI)/a)+n},OutElastic:function(t,n,e,i){var r=1.70158,a=0,o=e;if(0==t)return n;if(1==(t/=i))return n+e;if(a||(a=.3*i),o<Math.abs(e)){o=e;r=a/4}else r=a/(2*Math.PI)*Math.asin(e/o);return o*Math.pow(2,-10*t)*Math.sin((t*i-r)*(2*Math.PI)/a)+e+n},InOutElastic:function(t,n,e,i){var r=1.70158,a=0,o=e;if(0==t)return n;if(2==(t/=i/2))return n+e;if(a||(a=i*(.3*1.5)),o<Math.abs(e)){o=e;r=a/4}else r=a/(2*Math.PI)*Math.asin(e/o);return t<1?o*Math.pow(2,10*(t-=1))*Math.sin((t*i-r)*(2*Math.PI)/a)*-.5+n:o*Math.pow(2,-10*(t-=1))*Math.sin((t*i-r)*(2*Math.PI)/a)*.5+e+n},InBack:function(t,n,e,i,r){return null==r&&(r=1.70158),e*(t/=i)*t*((r+1)*t-r)+n},OutBack:function(t,n,e,i,r){return null==r&&(r=1.70158),e*((t=t/i-1)*t*((r+1)*t+r)+1)+n},InOutBack:function(t,n,e,i,r){return null==r&&(r=1.70158),(t/=i/2)<1?e/2*(t*t*((1+(r*=1.525))*t-r))+n:e/2*((t-=2)*t*((1+(r*=1.525))*t+r)+2)+n},InBounce:function(t,n,e,i){return e-this.OutBounce(i-t,0,e,i)+n},OutBounce:function(t,n,e,i){return(t/=i)<1/2.75?e*(7.5625*t*t)+n:t<2/2.75?e*(7.5625*(t-=1.5/2.75)*t+.75)+n:t<2.5/2.75?e*(7.5625*(t-=2.25/2.75)*t+.9375)+n:e*(7.5625*(t-=2.625/2.75)*t+.984375)+n},InOutBounce:function(t,n,e,i){return t<i/2?.5*this.InBounce(2*t,0,e,i)+n:.5*this.OutBounce(2*t-i,0,e,i)+.5*e+n}}}.Run(t)};
window.OnlineWebFonts_Com=window.OnlineWebFonts_Com||function(t){return new OnlineWebFonts_Animations(t);};if(typeof Object.assign != 'function'){Object.assign = function(e){e = Object(e);for(var i=1;i<arguments.length;i++){var s=arguments[i];if(s != null){for(var k in s){if(Object.prototype.hasOwnProperty.call(s,k)){e[k] = s[k];}}}}return e;}}
window.__Animations = Object.assign(window.__Animations || {},{"432926":{"Line":[{"Path":"M115.3,13.2C55.2,19.9,10,68.9,10,127.7c0,13.7,1.6,24.3,5.7,36.1c6.1,18.2,15.9,33.4,30,46.9c40.2,38.4,102.1,43.7,148.6,12.8c21.7-14.5,37.7-35.2,45.8-59.3c15.8-47.2-0.6-98.2-41.2-128.3C175.8,18.7,143.7,10,115.3,13.2z M141.1,28.2c53.1,6.7,92.5,52.9,89.6,105.1c-1.4,25.1-11.9,48-30,65.6c-13.3,12.9-28.9,21.8-46.6,26.6c-8.5,2.3-11.1,2.6-26,2.6c-14.5,0.1-17.6-0.2-25-2.2c-37.4-9.8-65.3-37-75.3-73.6c-1.7-6-2.1-10-2.1-23.8c0-14.3,0.3-17.6,2.1-24.2C36,73.8,55.4,50.7,84.1,37.1C101.6,28.8,121.3,25.8,141.1,28.2z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M74.3,88.7c-2.8,3.7-5.4,6.6-5.8,6.6c-0.3,0-1.8-1.3-3.3-2.8c-1.5-1.6-3.2-2.8-3.8-2.8c-1.3,0-5.2,4-5.2,5.3c0,0.5,2.9,3.8,6.5,7.4l6.4,6.4l7.8-9.8c4.3-5.3,8.3-10.3,8.8-11c1.1-1.4-2.6-5.9-4.9-5.9C80.1,82.2,77.2,85.1,74.3,88.7z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M97.5,97.4c-2,2.7-1.9,4.8,0.2,6.7c1.6,1.4,6.2,1.6,46.8,1.6c39.6,0,45.1-0.2,46.4-1.5c2-2,1.9-5.4-0.2-7.3c-1.6-1.4-6.2-1.6-46.8-1.6h-45L97.5,97.4z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M74.2,122.3l-5.9,7.2l-3.2-3.5c-2.9-3.2-3.3-3.4-5.2-2.2c-4.5,3-4.3,4,2.1,10.9c3.3,3.5,6.4,6.5,7,6.6c0.9,0.2,14.6-16.2,17-20.3c0.6-1,0.2-2.1-1.7-3.7c-1.4-1.2-2.9-2.2-3.4-2.2S77.4,118.4,74.2,122.3z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M97.9,126.5c-2.4,2.4-2.5,5.1,0,7.6c1.9,1.9,3.1,1.9,52.9,1.9s51,0,52.9-1.9c2.3-2.3,2.4-4.6,0.6-7.2c-1.3-1.7-3.1-1.8-52.9-2.1C100.7,124.6,99.8,124.6,97.9,126.5z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M74.2,155.4l-5.8,7.1l-3.1-3.3c-1.7-1.9-3.5-3.4-4-3.4c-1.3,0-5.3,4.4-4.9,5.5c0.6,1.4,12.1,13.4,12.8,13.4c0.4-0.1,2.9-2.9,5.7-6.4c2.6-3.5,6.4-8.4,8.4-11l3.6-4.5l-2.5-2.3c-1.5-1.2-3-2.3-3.5-2.3C80.4,148.2,77.4,151.4,74.2,155.4z","Duration":0,"Width":"3","Color":"#000000"},{"Path":"M97.6,156.5c-2.1,2.3-2,5.2,0.1,7.1c1.6,1.4,5.8,1.6,41,1.6c36.8,0,39.3-0.1,40.7-1.7c2.1-2.3,2-5.2-0.1-7.1c-1.6-1.4-5.8-1.6-41-1.6C101.6,154.8,99,154.9,97.6,156.5z","Duration":0,"Width":"3","Color":"#000000"}],"Box":{"Width":"256","Height":"256"},"Config":{"Width":"3","Opacity":1,"Sequential":true,"Color":"#000000","Animate":"Linear","Reverse":false}}});