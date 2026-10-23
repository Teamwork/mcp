#!/usr/bin/env python3
"""Encode preserved real VHS frames when VHS 0.12.0 skips video export."""
import argparse
from pathlib import Path
import struct
import subprocess


def frames(directory, stream):
    paths = sorted(directory.glob(f'frame-{stream}-*.png'))
    numbers = [int(path.stem.rsplit('-', 1)[1]) for path in paths]
    if not numbers or numbers != list(range(1, numbers[-1] + 1)):
        raise ValueError(f'{stream}: empty stream or missing frame numbers; no frames have been substituted')
    for path in paths:
        with path.open('rb') as handle:
            header = handle.read(24)
            handle.seek(-12, 2)
            end = handle.read()
        if header[:8] != b'\x89PNG\r\n\x1a\n' or end != b'\0\0\0\0IEND\xaeB`\x82':
            raise ValueError(f'incomplete PNG: {path.name}')
    return len(paths), struct.unpack('>II', header[16:24])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--frames', type=Path, default=Path('teamwork-review-frames'))
    parser.add_argument('--output', type=Path, default=Path('teamwork-review'), help='output basename without extension')
    args = parser.parse_args()
    text_count, text_dimensions = frames(args.frames, 'text')
    cursor_count, cursor_dimensions = frames(args.frames, 'cursor')
    if text_count != cursor_count or text_dimensions != cursor_dimensions:
        raise ValueError('stream counts/dimensions differ; wait for capture to finish')
    # These settings match demo.tape. Keep them synchronized when editing it.
    fps, width, height, padding, background = 15, 1600, 1000, 24, '#1e1e2e'
    print(f'Encoding {text_count} matched, contiguous real frames per stream; duration {text_count / fps:.2f}s.', flush=True)
    filters = (f'[0][1]overlay=shortest=1[merged];'
               f'[merged]scale={width - 2 * padding}:{height - 2 * padding}:force_original_aspect_ratio=1[scaled];'
               f'[scaled]fps={fps},setpts=PTS/1.0[speed];'
               f'[speed]pad={width}:{height}:(ow-iw)/2:(oh-ih)/2:{background}[padded];'
               f'[padded]fillborders=left={padding}:right={padding}:top={padding}:bottom={padding}:'
               f'mode=fixed:color={background}[out]')
    args.output.parent.mkdir(parents=True, exist_ok=True)
    base = ['ffmpeg', '-hide_banner', '-loglevel', 'warning', '-xerror', '-y',
            '-threads', '2', '-r', str(fps), '-start_number', '1', '-i', str(args.frames / 'frame-text-%05d.png'),
            '-threads', '2', '-r', str(fps), '-start_number', '1', '-i', str(args.frames / 'frame-cursor-%05d.png'),
            '-filter_complex_threads', '2', '-filter_complex', filters, '-map', '[out]',
            '-an', '-pix_fmt', 'yuv420p', '-threads', '2']
    outputs = [('.mp4', ['-c:v', 'libx264', '-crf', '20', '-movflags', '+faststart']),
               ('.webm', ['-c:v', 'libvpx-vp9', '-crf', '30', '-b:v', '0'])]
    for extension, options in outputs:
        target = Path(str(args.output) + extension)
        subprocess.run(base + options + [str(target)], check=True)
        if not target.is_file() or target.stat().st_size == 0:
            raise RuntimeError(f'encoder did not produce {target}')
        subprocess.run(['ffprobe', '-v', 'error', '-show_entries', 'format=duration,size',
                        '-show_entries', 'stream=codec_name,width,height', '-of', 'json', str(target)], check=True)
        print(f'Created {target.resolve()}', flush=True)


if __name__ == '__main__':
    main()
