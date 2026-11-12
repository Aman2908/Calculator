import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import App from './App';
import { ApiError } from './api/client';
import { fakeClient } from './test/fakeClient';

function setup(client = fakeClient()) {
  const user = userEvent.setup();
  render(<App client={client} />);
  return { user, client };
}

const result = () => screen.getByRole('region', { name: 'Result' });

describe('App', () => {
  it('adds two numbers and shows the expression and result', async () => {
    const { user, client } = setup();

    await user.type(screen.getByLabelText('First number'), '2');
    await user.type(screen.getByLabelText('Second number'), '3');
    await user.click(screen.getByRole('button', { name: 'Calculate' }));

    expect(await screen.findByTestId('result')).toHaveTextContent('5');
    expect(result()).toHaveTextContent('2 + 3 =');
    expect(client.calculate).toHaveBeenCalledWith(
      { operation: 'add', operands: [2, 3] },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });

  it('submits with the Enter key', async () => {
    const { user } = setup();
    await user.click(screen.getByRole('radio', { name: 'Multiply' }));
    await user.type(screen.getByLabelText('First number'), '6');
    await user.type(screen.getByLabelText('Second number'), '7{Enter}');
    expect(await screen.findByTestId('result')).toHaveTextContent('42');
  });

  it('hides the second field for square root', async () => {
    const { user } = setup();
    await user.click(screen.getByRole('radio', { name: 'Square root' }));

    expect(screen.queryByLabelText('Second number')).not.toBeInTheDocument();
    await user.type(screen.getByLabelText('Number'), '81');
    await user.click(screen.getByRole('button', { name: 'Calculate' }));

    expect(await screen.findByTestId('result')).toHaveTextContent('9');
    expect(result()).toHaveTextContent('√81 =');
  });

  it('shows field errors without calling the server', async () => {
    const { user, client } = setup();
    await user.type(screen.getByLabelText('First number'), 'abc');
    await user.click(screen.getByRole('button', { name: 'Calculate' }));

    const a = screen.getByLabelText('First number');
    expect(a).toHaveAttribute('aria-invalid', 'true');
    expect(a).toHaveAccessibleDescription(/valid number/);
    expect(screen.getByLabelText('Second number')).toHaveAccessibleDescription('Enter a number.');
    expect(client.calculate).not.toHaveBeenCalled();

    // Typing into the field clears its error.
    await user.type(a, '1');
    expect(a).not.toHaveAttribute('aria-invalid');
  });

  it('pre-empts division by zero on the client', async () => {
    const { user, client } = setup();
    await user.click(screen.getByRole('radio', { name: 'Divide' }));
    await user.type(screen.getByLabelText('First number'), '1');
    await user.type(screen.getByLabelText('Second number'), '0');
    await user.click(screen.getByRole('button', { name: 'Calculate' }));

    expect(screen.getByText('Cannot divide by zero.')).toBeInTheDocument();
    expect(client.calculate).not.toHaveBeenCalled();
  });

  it('announces a 422 from the server without a retry button', async () => {
    const client = fakeClient({
      calculate: vi.fn().mockRejectedValue(new ApiError('NON_FINITE_RESULT', 'result is too large', 422)),
    });
    const { user } = setup(client);
    await user.click(screen.getByRole('radio', { name: 'Power' }));
    await user.type(screen.getByLabelText('Base'), '10');
    await user.type(screen.getByLabelText('Exponent'), '400');
    await user.click(screen.getByRole('button', { name: 'Calculate' }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent("Can't calculate that");
    expect(alert).toHaveTextContent('result is too large');
    expect(within(alert).queryByRole('button')).not.toBeInTheDocument();
  });

  it('labels 400 errors as invalid input', async () => {
    const client = fakeClient({
      calculate: vi.fn().mockRejectedValue(new ApiError('INVALID_OPERANDS', 'bad operands', 400)),
    });
    const { user } = setup(client);
    await user.type(screen.getByLabelText('First number'), '1');
    await user.type(screen.getByLabelText('Second number'), '2{Enter}');
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid input');
  });

  it('offers a retry on network errors and retries the same request', async () => {
    const calculate = vi
      .fn()
      .mockRejectedValueOnce(new ApiError('NETWORK_ERROR', 'Could not reach the server.', 0))
      .mockResolvedValueOnce({ operation: 'add', operands: [1, 2], result: 3 });
    const { user } = setup(fakeClient({ calculate }));
    await user.type(screen.getByLabelText('First number'), '1');
    await user.type(screen.getByLabelText('Second number'), '2{Enter}');

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Connection problem');
    await user.click(within(alert).getByRole('button', { name: 'Try again' }));

    expect(await screen.findByTestId('result')).toHaveTextContent('3');
    expect(calculate).toHaveBeenCalledTimes(2);
    expect(calculate.mock.calls[1]?.[0]).toEqual({ operation: 'add', operands: [1, 2] });
  });

  it('labels other server failures generically', async () => {
    const client = fakeClient({
      calculate: vi.fn().mockRejectedValue(new ApiError('INTERNAL', 'internal server error', 500)),
    });
    const { user } = setup(client);
    await user.type(screen.getByLabelText('First number'), '1');
    await user.type(screen.getByLabelText('Second number'), '2{Enter}');
    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong');
  });

  it('refills the form from history', async () => {
    const { user } = setup();
    await user.click(screen.getByRole('radio', { name: 'Percent (b% of a)' }));
    await user.type(screen.getByLabelText('Amount'), '200');
    await user.type(screen.getByLabelText('Percent'), '15{Enter}');
    expect(await screen.findByTestId('result')).toHaveTextContent('30');

    await user.click(screen.getByRole('radio', { name: 'Add' }));
    await user.clear(screen.getByLabelText('First number'));

    await user.click(screen.getByRole('button', { name: 'Reuse 15% of 200' }));
    expect(screen.getByRole('radio', { name: 'Percent (b% of a)' })).toBeChecked();
    expect(screen.getByLabelText('Amount')).toHaveValue('200');
    expect(screen.getByLabelText('Percent')).toHaveValue('15');
  });

  it('uses the server operation list when it loads', async () => {
    const client = fakeClient({
      listOperations: vi.fn().mockResolvedValue([{ name: 'add', symbol: '+', arity: 2, label: 'Add' }]),
    });
    setup(client);
    await waitFor(() => expect(screen.getAllByRole('radio')).toHaveLength(1));
  });

  it('keeps the fallback operation list when loading fails', async () => {
    const listOperations = vi.fn().mockRejectedValue(new ApiError('NETWORK_ERROR', 'down', 0));
    setup(fakeClient({ listOperations }));
    await waitFor(() => expect(listOperations).toHaveBeenCalled());
    expect(screen.getAllByRole('radio')).toHaveLength(7);
  });

  it('disables the button while a request is in flight', async () => {
    const calculate = vi.fn(() => new Promise<never>(() => {}));
    const { user } = setup(fakeClient({ calculate }));
    await user.type(screen.getByLabelText('First number'), '1');
    await user.type(screen.getByLabelText('Second number'), '2{Enter}');
    expect(screen.getByRole('button', { name: 'Calculating…' })).toBeDisabled();
    expect(result()).toHaveTextContent('Calculating…');
  });
});
