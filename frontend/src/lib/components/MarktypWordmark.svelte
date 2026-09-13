<script lang="ts">
  import { marktypWordmarkPixels } from '../utils/asciiArt';

  export let size = 24;
  export let color = 'currentColor';
  export let className = '';
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  width={size * (41 / 7)}
  height={size}
  viewBox="0 0 41 7"
  fill={color}
  class={className}
  aria-label="Marktyp"
  role="img"
>
  {#each marktypWordmarkPixels as pixel (pixel.x + '-' + pixel.y)}
    <rect x={pixel.x} y={pixel.y} width="1" height="1" />
  {/each}
</svg>
