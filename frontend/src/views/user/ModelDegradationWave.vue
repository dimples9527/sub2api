<script setup lang="ts">
// 封面底纹用的信号波形：纯装饰（aria-hidden），颜色由外层容器的 --ma-glow 传下来，
// 所以判定色 / 明暗主题都不用在这里管。
withDefaults(defineProps<{ running?: boolean }>(), { running: false })
</script>

<template>
  <svg
    class="ma-wave"
    :class="{ 'is-running': running }"
    viewBox="0 0 320 200"
    preserveAspectRatio="none"
    aria-hidden="true"
    focusable="false"
  >
    <path
      class="ma-wave-line ma-wave-line-far"
      d="M0 100 C 20 74, 60 74, 80 100 S 140 126, 160 100 S 220 74, 240 100 S 300 126, 320 100"
    />
    <path
      class="ma-wave-line ma-wave-line-mid"
      d="M0 100 C 24 58, 56 142, 80 100 S 136 58, 160 100 S 216 142, 240 100 S 296 58, 320 100"
    />
    <path
      class="ma-wave-line ma-wave-line-near"
      d="M0 100 C 28 44, 52 156, 80 100 S 132 44, 160 100 S 212 156, 240 100 S 292 44, 320 100"
    />
  </svg>
</template>

<style scoped>
/* 跟 CSS 那两层光晕 / 网格同层（z-index:-1）：在封面底色之上、文字与缩略图之下。
   父级 .ma-*-stage 带 isolation: isolate，负层级才不会掉出卡片。 */
.ma-wave {
  position: absolute;
  inset: 0;
  z-index: -1;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.ma-wave-line {
  fill: none;
  stroke: var(--ma-glow, currentColor);
  stroke-width: 2;
  stroke-linecap: round;
  /* 容器是 preserveAspectRatio="none"，不锁的话横线会被压成几像素粗的实块。 */
  vector-effect: non-scaling-stroke;
  stroke-dasharray: 30 22;
  animation: ma-wave-flow 9s linear infinite;
}

/* 三条同形不同振幅：远的那条慢而淡，近的那条快而粗，叠出点纵深。 */
.ma-wave-line-far {
  opacity: 0.3;
  stroke-width: 1.5;
  animation-duration: 15s;
  animation-direction: reverse;
}

.ma-wave-line-mid {
  opacity: 0.24;
  animation-duration: 11s;
}

.ma-wave-line-near {
  opacity: 0.18;
  stroke-width: 3;
  animation-duration: 7s;
}

/* 跑着的那一轮把三条都提上来，一眼看得出还在动。 */
.ma-wave.is-running .ma-wave-line-far {
  opacity: 0.4;
  animation-duration: 4.4s;
}

.ma-wave.is-running .ma-wave-line-mid {
  opacity: 0.32;
  animation-duration: 3.2s;
}

.ma-wave.is-running .ma-wave-line-near {
  opacity: 0.26;
  animation-duration: 2.2s;
}

/* 52(30+22) 的整数倍，走完一圈正好接回起点，看不出跳帧。 */
@keyframes ma-wave-flow {
  to {
    stroke-dashoffset: -208;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ma-wave-line {
    animation: none;
  }
}
</style>
