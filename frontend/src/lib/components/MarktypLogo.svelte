<script lang="ts">
  import { marktypLogoPixels } from '../utils/asciiArt';

  export let size = 24;
  export let color = 'currentColor';
  export let className = '';
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  width={size}
  height={size}
  viewBox="0 0 7 7"
  fill={color}
  class={className}
  aria-label="Marktyp"
  role="img"
>
  {#each marktypLogoPixels as pixel (pixel.x + '-' + pixel.y)}
    <rect x={pixel.x} y={pixel.y} width="1" height="1" />
  {/each}
</svg>
