import json
import sys


VALIDATED_COMMANDS = [
    'remove',
    'priority'
]


def parse_har_file(har_file_path):
    with open(har_file_path, 'r') as har_file:
        har_file_json = json.load(har_file)
    return har_file_json


def save_har_file(har_file_object, har_file_path):
    with open(har_file_path, 'w') as target_file:
        target_file.write(json.dumps(har_file_object, indent=2))


def delete_content_field(har_file_json):
    entries = har_file_json['log']['entries']
    for entry in entries:
        if 'text' in entry['response']['content']:
            del entry['response']['content']['text']
    return har_file_json


def command_remove(argv):
    har_file_path = argv[1]
    output_file_name = argv[3]
    har_file_json = parse_har_file(har_file_path)
    content_field_removed = delete_content_field(har_file_json)
    save_har_file(content_field_removed, output_file_name)


def get_deps(item):
    """
    深度优先搜索，搜索到的所有依赖项都会被添加到 set 类型实例 deps 中。每一个 item
    对象均包含 callFrames 数组和 parent 对象。如果 parent 对象非空，则继续在
    parent 对象中递归寻找依赖项。如果不包含 parent 对象，则只需查找 callFrames
    对象中的内容。返回值是依赖项列表
    """
    result = set()
    # 查找 callFrames 中包含的依赖项
    call_frames = item['callFrames']
    deps_from_call_frames = set()
    for frame in call_frames:
        deps_from_call_frames.add(frame['url'])
    for dep in deps_from_call_frames:
        result.add(dep)
    # result.extend(list(deps_from_call_frames))

    if 'parent' in item:
        # 含有 parent 对象，需要在此对象中进行递归查找
        deps_from_parent = get_deps(item['parent'])
        for dep in deps_from_parent:
            result.add(dep)
        # result.extend(list(deps_from_parent))
    return list(result)


def generate_priority_tree(har_file_json):
    entries = har_file_json['log']['entries']
    # num_entries = len(entries)

    result = []

    root_request_url = ''

    # print('num_entries: %d' % num_entries)
    for entry in entries:
        extracted_entry_info = {}

        request = entry['request']
        # method = request['method']
        url = request['url']

        response = entry['response']
        status = response['status']
        # size = response['content']['size']
        # mime_type = response['content']['mimeType']
        # priority = entry['_priority']
        resource_type = entry['_resourceType']

        if root_request_url == '':
            root_request_url = url
            # print('root request found, url = <%s>' % url)
        initiator = entry['_initiator']

        extracted_entry_info['url'] = url
        extracted_entry_info['resource_type'] = resource_type

        if status != 404 and url.find('https://www.stormlin.com') >= 0:
            # print('url = <%s>, priority = <%s>, resource_type = <%s>'
            #       % (url, priority, resource_type))
            has_stack = 'stack' in initiator
            initiator_type = initiator['type']
            # print('url = <%s>, type = <%s>, stack = <%s>' % (url, intiator['type'], has_stack))
            # print('extracting deps for url = <%s>, type = <%s>, has_stack = <%s>' % (url, initiator_type, has_stack))
            extracted_entry_info['initiator_type'] = initiator_type
            # print('generate_priority_tree: %s, initiator_type = <%s>' % (url, initiator_type))
            if initiator_type == 'parser' or initiator_type == 'other':
                # print('  0. %s' % root_request_url)
                extracted_entry_info['deps'] = [root_request_url]
                result.append(extracted_entry_info)
                continue
            if has_stack:
                # find all deps by deep first search
                deps_list = get_deps(initiator['stack'])
                extracted_entry_info['deps'] = deps_list
                result.append(extracted_entry_info)

    return result


def command_priority(argv):
    har_file_path = argv[1]
    output_file_name = argv[3]
    har_file_json = parse_har_file(har_file_path)
    priority_info_list = generate_priority_tree(har_file_json)
    entry_count = 1
    for info in priority_info_list:
        resource_type = info['resource_type']
        if resource_type not in ['document', 'script', 'stylesheet']:
            continue
        print('%s. url = <%s>, resource_type = <%s>, initiator_type = <%s>' %
              (entry_count, info['url'], info['resource_type'], info['initiator_type']))
        dep_count = 1
        for dep in info['deps']:
            print('  %s. url = <%s>' % (dep_count, dep))
            dep_count += 1
        entry_count += 1


def main(argv):
    # parse arguments from command line
    if len(argv) <= 0:
        print('error: no command provided')
        return

    # extract command to be executed
    command = argv[0]
    if not (command in VALIDATED_COMMANDS):
        print('error: unsupported command <%s>' % command)
        return

    # execute command
    if command == 'remove':
        command_remove(argv[1:])
        return
    elif command == 'priority':
        command_priority(argv[1:])
        return


if __name__ == "__main__":
    main(sys.argv[1:])
