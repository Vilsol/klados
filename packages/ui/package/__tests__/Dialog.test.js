import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import DialogTest from './DialogTest.svelte';
describe('Dialog', () => {
    it('renders trigger in closed state without throwing', () => {
        const { getByText } = render(DialogTest);
        expect(getByText('Open')).toBeTruthy();
    });
});
