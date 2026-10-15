<script setup lang="ts">
// 检测吉祥物：一只在思考的小狗 / 一只抓耳挠腮的小猴子，按判定换表情。
// 纯装饰，不进无障碍树；毛色固定暖色 —— 跟主题明暗走会变得很怪。
type MascotAnimal = 'dog' | 'monkey'
type MascotState = 'thinking' | 'pass' | 'fail' | 'unknown'

withDefaults(defineProps<{ animal?: MascotAnimal; state?: MascotState }>(), {
  animal: 'dog',
  state: 'thinking',
})
</script>

<template>
  <svg
    class="mc"
    :class="[`is-${animal}`, `is-${state}`]"
    viewBox="0 0 120 120"
    aria-hidden="true"
    focusable="false"
  >
    <!-- 头顶那圈：think 转三个点、pass 冒星星、fail 顶个问号、unknown 顶个省略 -->
    <g v-if="state === 'thinking'" class="mc-think">
      <circle class="mc-dot mc-dot-1" cx="34" cy="26" r="4" />
      <circle class="mc-dot mc-dot-2" cx="48" cy="17" r="5.6" />
      <circle class="mc-dot mc-dot-3" cx="65" cy="12" r="7.2" />
    </g>
    <g v-else-if="state === 'pass'" class="mc-sparks">
      <path
        class="mc-spark mc-spark-1"
        d="M34 20 C35 26 38 29 44 30 C38 31 35 34 34 40 C33 34 30 31 24 30 C30 29 33 26 34 20 Z"
      />
      <path
        class="mc-spark mc-spark-2"
        d="M84 26 C85 31 87 33 92 34 C87 35 85 37 84 42 C83 37 81 35 76 34 C81 33 83 31 84 26 Z"
      />
    </g>
    <text v-else-if="state === 'fail'" class="mc-mark" x="82" y="30">?</text>
    <text v-else class="mc-mark mc-mark-dim" x="82" y="30">…</text>

    <g class="mc-body-group">
      <!-- ================= 小狗（垂耳 + 长鼻吻 + 趴脸爪） ================= -->
      <template v-if="animal === 'dog'">
        <!-- 尾巴往右侧平伸出去，别往头顶翘 —— 翘起来就跟猴子那只挠头手撞形了 -->
        <path class="mc-tail" d="M80 94 C 94 98 102 94 104 88 C 106 83 102 80 98 83" />
        <ellipse class="mc-fur" cx="60" cy="90" rx="24" ry="19" />
        <ellipse class="mc-fur-soft" cx="60" cy="98" rx="14" ry="11" />
        <ellipse class="mc-paw-sole" cx="48" cy="106" rx="7" ry="4.5" />
        <ellipse class="mc-paw-sole" cx="72" cy="106" rx="7" ry="4.5" />
        <!-- 项圈：脖子上一圈加个小牌，一眼认出是狗 -->
        <path class="mc-collar" d="M42 75 C 50 81 70 81 78 75" />
        <circle class="mc-tag" cx="60" cy="80" r="3.6" />

        <!-- 垂耳：旋转一下让它真「垂」下来，与猴子的圆耳一眼分得开 -->
        <ellipse class="mc-ear mc-ear-l" cx="35" cy="51" rx="7" ry="17" transform="rotate(-8 35 39)" />
        <ellipse class="mc-ear mc-ear-r" cx="85" cy="51" rx="7" ry="17" transform="rotate(8 85 39)" />
        <circle class="mc-fur" cx="60" cy="50" r="24" />
        <ellipse class="mc-fur-soft" cx="60" cy="64" rx="14" ry="11" />

        <!-- 眼睛：think/unknown 是圆的，pass 弯成月牙，fail 眯成缝 -->
        <template v-if="state === 'pass'">
          <path class="mc-eye-happy" d="M44 46 C 47 40 53 40 56 46" />
          <path class="mc-eye-happy" d="M64 46 C 67 40 73 40 76 46" />
        </template>
        <template v-else-if="state === 'fail'">
          <path class="mc-eye-squint" d="M44 46 L 54 49" />
          <path class="mc-eye-squint" d="M76 46 L 66 49" />
        </template>
        <template v-else>
          <circle class="mc-eye" cx="49" cy="46" r="3.4" />
          <circle class="mc-eye" cx="71" cy="46" r="3.4" />
        </template>

        <ellipse class="mc-nose" cx="60" cy="58" rx="5" ry="3.8" />
        <path class="mc-mouth" d="M60 61 L 60 65 M 60 65 C 56 70 51 70 48 66" />
        <path class="mc-mouth" d="M60 65 C 64 70 69 70 72 66" />

        <!-- 思考爪搭下巴：比脸深一档的毛色，不然同色叠在脸上等于没有 -->
        <ellipse class="mc-paw" cx="73" cy="73" rx="8.5" ry="6.4" />
        <ellipse class="mc-fur-soft" cx="73" cy="73" rx="4.2" ry="2.8" />

        <g v-if="state === 'fail'" class="mc-sweat">
          <path d="M88 46 C 92 52 93 55 90 57 C 87 59 84 57 84 54 C 84 51 86 49 88 46 Z" />
        </g>
      </template>

      <!-- ================= 小猴子（圆耳 + 挠头手） ================= -->
      <template v-else>
        <path class="mc-tail" d="M82 96 C 102 100 112 86 106 74 C 102 66 94 66 94 72 C 94 76 98 76 98 73" />
        <ellipse class="mc-fur" cx="60" cy="88" rx="22" ry="22" />
        <ellipse class="mc-fur-soft" cx="60" cy="94" rx="13" ry="14" />
        <ellipse class="mc-paw-sole" cx="50" cy="106" rx="7" ry="4.5" />
        <ellipse class="mc-paw-sole" cx="70" cy="106" rx="7" ry="4.5" />

        <!-- 圆耳比脸还鼓一点 -->
        <circle class="mc-ear mc-ear-l" cx="34" cy="48" r="10" />
        <circle class="mc-ear mc-ear-r" cx="86" cy="48" r="10" />
        <circle class="mc-fur" cx="60" cy="50" r="24" />
        <ellipse class="mc-fur-soft" cx="60" cy="54" rx="16" ry="14" />

        <template v-if="state === 'pass'">
          <path class="mc-eye-happy" d="M44 46 C 47 40 53 40 56 46" />
          <path class="mc-eye-happy" d="M64 46 C 67 40 73 40 76 46" />
        </template>
        <template v-else-if="state === 'fail'">
          <path class="mc-eye-squint" d="M44 46 L 54 49" />
          <path class="mc-eye-squint" d="M76 46 L 66 49" />
        </template>
        <template v-else>
          <circle class="mc-eye" cx="52" cy="46" r="3.2" />
          <circle class="mc-eye" cx="68" cy="46" r="3.2" />
        </template>

        <circle class="mc-nose" cx="56" cy="54" r="1.4" />
        <circle class="mc-nose" cx="64" cy="54" r="1.4" />
        <path class="mc-mouth" d="M52 60 C 57 66 63 66 68 60" />

        <!-- 挠头手：从身体右上举到头顶，粗黑条让动作一目了然 -->
        <path class="mc-arm" d="M76 78 C 86 62 80 38 70 28" />

        <g v-if="state === 'fail'" class="mc-sweat">
          <path d="M88 44 C 92 50 93 53 90 55 C 87 57 84 55 84 52 C 84 49 86 47 88 44 Z" />
        </g>
      </template>
    </g>
  </svg>
</template>

<style scoped>
.mc {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.mc-fur {
  fill: #c98f52;
}

.mc-fur-soft {
  fill: #f5e2c8;
}

.mc-ear {
  fill: #a9703a;
}

.mc-paw-sole {
  fill: #b57c45;
}

.mc-nose {
  fill: #3b2a20;
}

.mc-eye {
  fill: #3b2a20;
}

.mc-mouth,
.mc-eye-happy,
.mc-eye-squint,
.mc-tail,
.mc-arm {
  fill: none;
  stroke: #3b2a20;
  stroke-width: 2.4;
  stroke-linecap: round;
}

/* 狗那只搭下巴的爪子是实心的，不能用 .mc-arm（那条规则是给猴子描边用的）。 */
.mc-paw {
  fill: #a9703a;
}

.mc-collar {
  fill: none;
  stroke: #4a7fb5;
  stroke-width: 4;
  stroke-linecap: round;
}

.mc-tag {
  fill: #e4bc50;
  stroke: #c99a2e;
  stroke-width: 1;
}

.is-monkey .mc-paw {
  fill: #a06a41;
}

.is-dog .mc-tail {
  stroke: #a9703a;
  stroke-width: 4.5;
}

.is-monkey .mc-fur {
  fill: #b57a4e;
}

.is-monkey .mc-fur-soft {
  fill: #f2d7b0;
}

.is-monkey .mc-ear {
  fill: #e8bd93;
}

.is-monkey .mc-paw-sole {
  fill: #a06a41;
}

.is-monkey .mc-arm {
  stroke: #a06a41;
  stroke-width: 7;
}

.is-monkey .mc-tail {
  stroke: #a06a41;
  stroke-width: 5;
}

.mc-mark {
  fill: #8a5f3c;
  font-size: 26px;
  font-weight: 700;
}

.mc-mark-dim {
  opacity: 0.45;
}

.mc-think .mc-dot {
  fill: #a9703a;
}

.mc-spark {
  fill: #e4bc50;
}

.mc-sweat path {
  fill: #7fb6dd;
}

/* 所有变换都锚在自身包围盒上 —— SVG 里没有 transform-box: fill-box 的话
   transform-origin 会按整个 viewBox 算，尾巴 / 耳朵会绕着画布中心甩。 */
.mc-body-group,
.mc-ear,
.mc-tail,
.mc-arm,
.mc-paw,
.mc-eye,
.mc-dot,
.mc-spark,
.mc-sweat {
  transform-box: fill-box;
}

.mc-body-group {
  transform-origin: 50% 100%;
  animation: mc-breathe 2.8s ease-in-out infinite;
}

/* 耳朵绕耳根抖，尾巴绕尾根摇 */
.mc-ear-l {
  transform-origin: 50% 0%;
  animation: mc-ear-twitch 3.4s ease-in-out infinite;
}

.mc-ear-r {
  transform-origin: 50% 0%;
  animation: mc-ear-twitch 3.4s ease-in-out infinite 0.4s;
}

.mc-tail {
  transform-origin: 0% 100%;
  animation: mc-tail-wag 1.5s ease-in-out infinite;
}

/* 思考的那只手 / 爪子：一下一下地蹭下巴、挠头 */
.mc-arm,
.mc-paw {
  transform-origin: 50% 100%;
  animation: mc-arm-think 1.7s ease-in-out infinite;
}

/* 眨眼：绝大部分时间睁着，偶尔压扁一下 */
.mc-eye {
  transform-origin: 50% 50%;
  animation: mc-blink 5.2s ease-in-out infinite;
}

.mc-dot {
  transform-origin: 50% 50%;
  animation: mc-dot-pop 1.5s ease-in-out infinite;
}

.mc-dot-2 {
  animation-delay: 0.18s;
}

.mc-dot-3 {
  animation-delay: 0.36s;
}

.mc-spark {
  transform-origin: 50% 50%;
  animation: mc-spark-twinkle 1.4s ease-in-out infinite;
}

.mc-spark-2 {
  animation-delay: 0.5s;
}

.mc-sweat {
  animation: mc-sweat-drop 1.9s ease-in-out infinite;
}

/* ---- 判定态 ---- */

/* 通过：整只轻轻跳一下，尾巴摇得欢 */
.is-pass .mc-body-group {
  animation: mc-hop 0.9s ease-in-out infinite;
}

.is-pass .mc-tail {
  animation-duration: 0.42s;
}

.is-pass .mc-arm,
.is-pass .mc-paw {
  animation: mc-arm-cheer 0.9s ease-in-out infinite;
}

/* 降智：耳朵耷下来、尾巴慢摆、脑袋有点懵地晃 */
.is-fail .mc-ear-l,
.is-fail .mc-ear-r {
  animation: none;
  transform: rotate(16deg);
}

.is-fail .mc-ear-r {
  transform: rotate(-16deg);
}

.is-fail .mc-tail {
  animation-duration: 3.2s;
}

.is-fail .mc-body-group {
  animation: mc-dizzy 2.6s ease-in-out infinite;
}

.is-fail .mc-arm,
.is-fail .mc-paw {
  animation-duration: 3s;
}

/* 未知 / 中止：手停住，只剩呼吸 */
.is-unknown .mc-arm,
.is-unknown .mc-paw {
  animation: none;
}

.is-unknown .mc-tail {
  animation-duration: 2.6s;
}

@keyframes mc-breathe {
  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.035, 0.972);
  }
}

@keyframes mc-hop {
  0%,
  100% {
    transform: translateY(0) scale(1);
  }

  35% {
    transform: translateY(-6px) scale(1.03, 0.97);
  }

  70% {
    transform: translateY(0) scale(0.985, 1.015);
  }
}

@keyframes mc-dizzy {
  0%,
  100% {
    transform: rotate(-2.5deg);
  }

  50% {
    transform: rotate(2.5deg);
  }
}

@keyframes mc-ear-twitch {
  0%,
  72%,
  100% {
    transform: rotate(0deg);
  }

  80% {
    transform: rotate(-9deg);
  }

  88% {
    transform: rotate(5deg);
  }
}

@keyframes mc-tail-wag {
  0%,
  100% {
    transform: rotate(-9deg);
  }

  50% {
    transform: rotate(11deg);
  }
}

@keyframes mc-arm-think {
  0%,
  100% {
    transform: rotate(-4deg);
  }

  50% {
    transform: rotate(5deg);
  }
}

@keyframes mc-arm-cheer {
  0%,
  100% {
    transform: rotate(-10deg) translateY(0);
  }

  50% {
    transform: rotate(8deg) translateY(-5px);
  }
}

@keyframes mc-blink {
  0%,
  90%,
  100% {
    transform: scaleY(1);
  }

  93% {
    transform: scaleY(0.12);
  }
}

@keyframes mc-dot-pop {
  0%,
  100% {
    opacity: 0.35;
    transform: scale(0.72);
  }

  40% {
    opacity: 1;
    transform: scale(1.12);
  }
}

@keyframes mc-spark-twinkle {
  0%,
  100% {
    opacity: 0.25;
    transform: scale(0.6) rotate(-12deg);
  }

  50% {
    opacity: 1;
    transform: scale(1.1) rotate(12deg);
  }
}

@keyframes mc-sweat-drop {
  0% {
    opacity: 0;
    transform: translateY(-6px) scale(0.7);
  }

  30% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }

  70%,
  100% {
    opacity: 0;
    transform: translateY(14px) scale(0.8);
  }
}

@media (prefers-reduced-motion: reduce) {
  .mc-body-group,
  .mc-ear,
  .mc-tail,
  .mc-arm,
  .mc-eye,
  .mc-dot,
  .mc-spark,
  .mc-sweat {
    animation: none;
  }
}
</style>
